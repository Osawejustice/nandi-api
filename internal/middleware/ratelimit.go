package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	count    int
	windowAt time.Time
}

// RateLimit is a simple per-IP sliding window used on auth endpoints.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 30
	}
	if window <= 0 {
		window = time.Minute
	}
	var mu sync.Mutex
	visitors := map[string]*visitor{}

	go func() {
		ticker := time.NewTicker(window)
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for k, v := range visitors {
				if now.Sub(v.windowAt) > window*2 {
					delete(visitors, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		mu.Lock()
		v, ok := visitors[ip]
		if !ok || now.Sub(v.windowAt) >= window {
			visitors[ip] = &visitor{count: 1, windowAt: now}
			mu.Unlock()
			c.Next()
			return
		}
		v.count++
		over := v.count > limit
		mu.Unlock()
		if over {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":       "rate_limited",
					"message":    "too many requests",
					"request_id": GetRequestID(c),
				},
			})
			return
		}
		c.Next()
	}
}
