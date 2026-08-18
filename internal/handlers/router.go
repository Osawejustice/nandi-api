package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/ai"
	"github.com/yourorg/nandi/internal/config"
	"github.com/yourorg/nandi/internal/middleware"
	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/providers"
	"github.com/yourorg/nandi/internal/realtime"
	"github.com/yourorg/nandi/internal/repositories"
	"github.com/yourorg/nandi/internal/services"
	"github.com/yourorg/nandi/internal/utils"
)

// Dependencies are infrastructure handles composed in main.
type Dependencies struct {
	DB     *gorm.DB
	Redis  *redis.Client
	Config *config.Config
	Log    zerolog.Logger
	Hub    *realtime.Hub
}

// Services is the wired application graph. Tests can construct a subset.
type Services struct {
	Auth      *services.AuthService
	Contacts  *services.ContactService
	Inbox     *services.InboxService
	Campaigns *services.CampaignService
	Analytics *services.AnalyticsService
	Router    *providers.Router
}

func BuildServices(deps Dependencies) Services {
	jwtm := utils.NewJWTManager(deps.Config.JWT)
	auth := services.NewAuthService(deps.DB, jwtm, deps.Log)
	contacts := services.NewContactService(deps.DB)

	var logRepo *repositories.ProviderLogRepo
	if deps.DB != nil {
		logRepo = repositories.NewProviderLogRepo(deps.DB)
	}

	at := providers.NewAfricaTalking(deps.Config.AT, deps.Log)
	evo := providers.NewEvolution(deps.Config.Evolution, deps.Log)
	stub := providers.NewStubProvider(deps.Log)
	registry := map[string]providers.ChannelProvider{
		"africastalking": at,
		"evolution":      evo,
		"stub":           stub,
	}
	router := providers.NewRouter(deps.Config.Provider, registry, logRepo, deps.Log)
	analyzer := ai.NewAnalyzer(deps.Config.AI, deps.Log)
	inbox := services.NewInboxService(deps.DB, contacts, router, deps.Hub, analyzer, deps.Log)
	campaigns := services.NewCampaignService(deps.DB, router, deps.Hub, deps.Log)
	analytics := services.NewAnalyticsService(deps.DB)

	return Services{
		Auth: auth, Contacts: contacts, Inbox: inbox,
		Campaigns: campaigns, Analytics: analytics, Router: router,
	}
}

// NewRouter builds the Gin engine with shared middleware and /api/v1 routes.
func NewRouter(log zerolog.Logger, deps Dependencies, svc Services) *gin.Engine {

	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.Logger(log),
		middleware.MaxBodyBytes(1<<20),
		middleware.CORS(deps.Config.CORS.Origins),
	)

	router.GET("/health", Health)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authH := NewAuthHandler(svc.Auth)
	keysH := NewAPIKeyHandler(svc.Auth)
	contactH := NewContactHandler(svc.Contacts)
	inboxH := NewConversationHandler(svc.Inbox)
	campH := NewCampaignHandler(svc.Campaigns)
	analyticsH := NewAnalyticsHandler(svc.Analytics, svc.Router)
	wsH := NewWSHandler(deps.Hub, svc.Auth, log)

	var tenantRepo *repositories.TenantRepo
	if deps.DB != nil {
		tenantRepo = repositories.NewTenantRepo(deps.DB)
	}
	hookH := NewWebhookHandler(svc.Inbox, tenantRepo, deps.Config.Webhook.Secret, log)

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.Use(middleware.RateLimit(40, 0))
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
			auth.POST("/refresh", authH.Refresh)
			auth.POST("/logout", authH.Logout)
		}

		v1.GET("/ws", wsH.Connect)

		hooks := v1.Group("/webhooks")
		{
			hooks.POST("/:tenant_slug/sms/africastalking", hookH.AfricaTalking)
			hooks.POST("/:tenant_slug/whatsapp/evolution", hookH.Evolution)
		}

		protected := v1.Group("")
		protected.Use(middleware.Authenticate(func(c *gin.Context, bearer, apiKey string) (*middleware.AuthPrincipal, error) {
			if bearer != "" {
				p, err := svc.Auth.AuthenticateJWT(bearer)
				if err != nil {
					return nil, err
				}
				return toPrincipal(p), nil
			}
			p, err := svc.Auth.AuthenticateAPIKey(c.Request.Context(), apiKey)
			if err != nil {
				return nil, err
			}
			return toPrincipal(p), nil
		}))
		{
			protected.GET("/me", authH.Me)
			protected.GET("/auth/me", authH.Me)

			protected.POST("/users", middleware.RequireRoles(models.RoleOwner, models.RoleAdmin), authH.CreateUser)

			keys := protected.Group("/api-keys")
			keys.Use(middleware.RequireRoles(models.RoleOwner, models.RoleAdmin))
			{
				keys.POST("", keysH.Create)
				keys.GET("", keysH.List)
				keys.DELETE("/:id", keysH.Revoke)
			}

			protected.GET("/contacts", contactH.List)
			protected.POST("/contacts", contactH.Create)
			protected.GET("/contacts/:id", contactH.Get)
			protected.PATCH("/contacts/:id", contactH.Update)
			protected.DELETE("/contacts/:id", contactH.Delete)

			protected.GET("/conversations", inboxH.List)
			protected.GET("/conversations/:id", inboxH.Get)
			protected.PATCH("/conversations/:id", inboxH.Update)
			protected.POST("/conversations/:id/messages", inboxH.Reply)
			protected.POST("/conversations/:id/summary", inboxH.Summarize)

			protected.GET("/campaigns", campH.List)
			protected.POST("/campaigns", campH.Create)
			protected.GET("/campaigns/:id", campH.Get)
			protected.POST("/campaigns/:id/start", campH.Start)

			protected.GET("/analytics/overview", analyticsH.Overview)
			protected.GET("/settings", analyticsH.GetSettings)
			protected.PUT("/settings", middleware.RequireRoles(models.RoleOwner, models.RoleAdmin), analyticsH.UpdateSettings)

			protected.GET("/agents", analyticsH.ListAgents)
			protected.POST("/agents/me/status", analyticsH.SetStatus)

			protected.POST("/dev/inbound", inboxH.SimulateInbound)
		}
	}

	router.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "not_found", "route not found")
	})

	return router
}

func toPrincipal(p *services.Principal) *middleware.AuthPrincipal {
	if p == nil {
		return nil
	}
	return &middleware.AuthPrincipal{
		UserID:   p.UserID,
		TenantID: p.TenantID,
		Role:     p.Role,
		AuthType: p.AuthType,
		APIKeyID: p.APIKeyID,
		HasUser:  p.HasUser,
	}
}
