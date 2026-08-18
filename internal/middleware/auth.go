package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Osawejustice/nandi-api/internal/models"
)

const (
	ContextUserID   = "auth_user_id"
	ContextTenantID = "auth_tenant_id"
	ContextRole     = "auth_role"
	ContextAuthType = "auth_type"
	ContextAPIKeyID = "auth_api_key_id"
	ContextHasUser  = "auth_has_user"
)

type AuthResolver func(c *gin.Context, bearer, apiKey string) (*AuthPrincipal, error)

type AuthPrincipal struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	Role     string
	AuthType string
	APIKeyID uuid.UUID
	HasUser  bool
}

// Authenticate accepts Authorization: Bearer <jwt> or X-API-Key.
func Authenticate(resolve AuthResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		bearer := extractBearer(c.GetHeader("Authorization"))
		apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if bearer == "" && apiKey == "" {
			abortUnauthorized(c, "missing credentials")
			return
		}

		principal, err := resolve(c, bearer, apiKey)
		if err != nil || principal == nil {
			abortUnauthorized(c, "invalid credentials")
			return
		}

		c.Set(ContextUserID, principal.UserID)
		c.Set(ContextTenantID, principal.TenantID)
		c.Set(ContextRole, principal.Role)
		c.Set(ContextAuthType, principal.AuthType)
		c.Set(ContextAPIKeyID, principal.APIKeyID)
		c.Set(ContextHasUser, principal.HasUser)
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role := Role(c)
		if _, ok := allowed[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":       "forbidden",
					"message":    "insufficient permissions",
					"request_id": GetRequestID(c),
				},
			})
			return
		}
		c.Next()
	}
}

func extractBearer(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":       "unauthorized",
			"message":    message,
			"request_id": GetRequestID(c),
		},
	})
}

func TenantID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ContextTenantID)
	id, _ := v.(uuid.UUID)
	return id
}

func UserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ContextUserID)
	id, _ := v.(uuid.UUID)
	return id
}

func Role(c *gin.Context) string {
	v, _ := c.Get(ContextRole)
	role, _ := v.(string)
	return role
}

func AuthType(c *gin.Context) string {
	v, _ := c.Get(ContextAuthType)
	t, _ := v.(string)
	if t == "" {
		return models.AuthTypeJWT
	}
	return t
}

func HasUser(c *gin.Context) bool {
	v, _ := c.Get(ContextHasUser)
	ok, _ := v.(bool)
	return ok
}

func MustUser(c *gin.Context) bool {
	return HasUser(c) && UserID(c) != uuid.Nil
}
