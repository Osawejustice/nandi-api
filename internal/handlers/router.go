package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"github.com/yourorg/nandi/internal/middleware"
)

// Dependencies are infrastructure handles composed in main. Unused on day 1
// beyond proving the connection helpers wire into the process.
type Dependencies struct {
	DB    *gorm.DB
	Redis *redis.Client
}

// NewRouter builds the Gin engine with shared middleware and day-1 routes.
func NewRouter(log zerolog.Logger, _ Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.Logger(log),
	)

	router.GET("/health", Health)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
