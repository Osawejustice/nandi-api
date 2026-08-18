package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the inbound/outbound request correlation header.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the Gin context key for the request ID.
	RequestIDKey = "request_id"
)

// RequestID assigns a request ID (honoring an inbound X-Request-ID when present).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDKey, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)
		c.Next()
	}
}

// GetRequestID returns the request ID stored on the Gin context.
func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(RequestIDKey); ok {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return ""
}
