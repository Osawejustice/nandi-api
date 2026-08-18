package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yourorg/nandi/internal/middleware"
	"github.com/yourorg/nandi/internal/services"
)

type CampaignHandler struct {
	svc *services.CampaignService
}

func NewCampaignHandler(svc *services.CampaignService) *CampaignHandler {
	return &CampaignHandler{svc: svc}
}

type CreateCampaignRequest struct {
	Name            string     `json:"name" binding:"required" example:"August promo"`
	Channel         string     `json:"channel" example:"sms"`
	MessageTemplate string     `json:"message_template" binding:"required" example:"Hello from Nandi"`
	Tag             string     `json:"tag" example:"vip"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
}

// List campaigns.
//
//	@Summary		List campaigns
//	@Tags			campaigns
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int	false	"Page"
//	@Param			per_page	query	int	false	"Page size"
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/campaigns [get]
func (h *CampaignHandler) List(c *gin.Context) {
	page, perPage := pageParams(c)
	items, total, err := h.svc.List(c.Request.Context(), middleware.TenantID(c), page, perPage)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writePage(c, items, total, page, perPage)
}

// Create campaign.
//
//	@Summary		Create campaign
//	@Tags			campaigns
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	CreateCampaignRequest	true	"Campaign"
//	@Success		201	{object}	map[string]any
//	@Router			/api/v1/campaigns [post]
func (h *CampaignHandler) Create(c *gin.Context) {
	var req CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	item, err := h.svc.Create(c.Request.Context(), middleware.TenantID(c), services.CreateCampaignInput{
		Name: req.Name, Channel: req.Channel, MessageTemplate: req.MessageTemplate,
		Tag: req.Tag, ScheduledAt: req.ScheduledAt, CreatedBy: middleware.UserID(c),
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusCreated, item)
}

// Get campaign.
//
//	@Summary		Get campaign
//	@Tags			campaigns
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Campaign ID"
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/campaigns/{id} [get]
func (h *CampaignHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), middleware.TenantID(c), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, item)
}

// Start queues a campaign for the worker.
//
//	@Summary		Start campaign
//	@Tags			campaigns
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Campaign ID"
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/campaigns/{id}/start [post]
func (h *CampaignHandler) Start(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	item, err := h.svc.Start(c.Request.Context(), middleware.TenantID(c), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, item)
}
