package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Osawejustice/nandi-api/internal/middleware"
	"github.com/Osawejustice/nandi-api/internal/models"
	"github.com/Osawejustice/nandi-api/internal/providers"
	"github.com/Osawejustice/nandi-api/internal/services"
)

type AnalyticsHandler struct {
	svc    *services.AnalyticsService
	router *providers.Router
}

func NewAnalyticsHandler(svc *services.AnalyticsService, router *providers.Router) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc, router: router}
}

// Overview metrics.
//
//	@Summary		Analytics overview
//	@Tags			analytics
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]services.Overview
//	@Router			/api/v1/analytics/overview [get]
func (h *AnalyticsHandler) Overview(c *gin.Context) {
	out, err := h.svc.Overview(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, out)
}

type SettingsRequest struct {
	FeatureFlags models.JSONMap `json:"feature_flags"`
	Preferences  models.JSONMap `json:"preferences"`
}

// Get settings.
//
//	@Summary		Get tenant settings
//	@Tags			settings
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/settings [get]
func (h *AnalyticsHandler) GetSettings(c *gin.Context) {
	row, err := h.svc.GetSettings(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	providers := map[string][]string{}
	if h.router != nil {
		providers = h.router.Status()
	}
	writeData(c, http.StatusOK, gin.H{
		"feature_flags": row.FeatureFlags,
		"preferences":   row.Preferences,
		"providers":     providers,
		"secrets_note":  "Provider credentials are configured via backend environment variables, never stored in the frontend.",
	})
}

// Update settings (flags/prefs only — no secrets).
//
//	@Summary		Update tenant settings
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	SettingsRequest	true	"Settings"
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/settings [put]
func (h *AnalyticsHandler) UpdateSettings(c *gin.Context) {
	var req SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	row, err := h.svc.UpdateSettings(c.Request.Context(), middleware.TenantID(c), req.FeatureFlags, req.Preferences)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, row)
}

type AgentStatusRequest struct {
	Status string `json:"status" binding:"required" example:"online"`
}

// List agents.
//
//	@Summary		List agents
//	@Tags			agents
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/agents [get]
func (h *AnalyticsHandler) ListAgents(c *gin.Context) {
	users, err := h.svc.ListAgents(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	out := make([]UserDTO, 0, len(users))
	for i := range users {
		out = append(out, userDTO(&users[i]))
	}
	writeData(c, http.StatusOK, out)
}

// Set my agent status.
//
//	@Summary		Set my agent status
//	@Tags			agents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	AgentStatusRequest	true	"Status"
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/agents/me/status [post]
func (h *AnalyticsHandler) SetStatus(c *gin.Context) {
	if !middleware.HasUser(c) {
		writeError(c, http.StatusForbidden, "forbidden", "API keys cannot set agent presence")
		return
	}
	var req AgentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	if err := h.svc.SetAgentStatus(c.Request.Context(), middleware.TenantID(c), middleware.UserID(c), req.Status); err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, gin.H{"status": req.Status})
}
