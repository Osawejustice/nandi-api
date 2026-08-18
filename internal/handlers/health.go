package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse is the payload returned by GET /health.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// Health returns a static liveness response.
//
//	@Summary		Health check
//	@Description	Returns service health status
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}
