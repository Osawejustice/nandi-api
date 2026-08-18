package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes rejects oversized JSON payloads. Webhooks stay under this too.
func MaxBodyBytes(n int64) gin.HandlerFunc {
	if n <= 0 {
		n = 1 << 20 // 1 MiB
	}
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		}
		c.Next()
	}
}
