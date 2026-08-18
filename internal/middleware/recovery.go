package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Recovery converts panics into a JSON 500 and logs the stack via zerolog.
func Recovery(log zerolog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Error().
			Str("request_id", GetRequestID(c)).
			Interface("panic", recovered).
			Str("path", c.Request.URL.Path).
			Msg("panic recovered")

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":       "internal_error",
				"message":    "internal server error",
				"request_id": GetRequestID(c),
			},
		})
	})
}
