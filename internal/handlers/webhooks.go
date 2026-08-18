package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/repositories"
	"github.com/yourorg/nandi/internal/services"
)

type WebhookHandler struct {
	inbox   *services.InboxService
	tenants *repositories.TenantRepo
	secret  string
	log     zerolog.Logger
}

func NewWebhookHandler(inbox *services.InboxService, tenants *repositories.TenantRepo, secret string, log zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{inbox: inbox, tenants: tenants, secret: secret, log: log}
}

func (h *WebhookHandler) authorize(c *gin.Context) bool {
	if h.secret == "" {
		return true
	}
	got := c.GetHeader("X-Webhook-Secret")
	if got == "" {
		got = c.Query("secret")
	}
	if got != h.secret {
		writeError(c, http.StatusUnauthorized, "unauthorized", "invalid webhook secret")
		return false
	}
	return true
}

func (h *WebhookHandler) resolveTenant(c *gin.Context) (*models.Tenant, bool) {
	if h.tenants == nil {
		writeError(c, http.StatusServiceUnavailable, "unavailable", "database is not available")
		return nil, false
	}
	slug := strings.TrimSpace(c.Param("tenant_slug"))
	if slug == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "missing tenant slug")
		return nil, false
	}
	tenant, err := h.tenants.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		writeError(c, http.StatusNotFound, "not_found", "unknown tenant")
		return nil, false
	}
	return tenant, true
}

// AfricaTalking inbound SMS.
//
//	@Summary		Africa's Talking inbound webhook
//	@Tags			webhooks
//	@Accept			mpfd
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			tenant_slug	path	string	true	"Tenant slug"
//	@Param			from		formData	string	true	"Sender"
//	@Param			text		formData	string	true	"Body"
//	@Param			id			formData	string	false	"Provider message id"
//	@Success		200	{object}	map[string]string
//	@Router			/api/v1/webhooks/{tenant_slug}/sms/africastalking [post]
func (h *WebhookHandler) AfricaTalking(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	tenant, ok := h.resolveTenant(c)
	if !ok {
		return
	}
	from := firstNonEmpty(c.PostForm("from"), c.Query("from"))
	text := firstNonEmpty(c.PostForm("text"), c.Query("text"))
	id := firstNonEmpty(c.PostForm("id"), c.Query("id"))
	if from == "" || text == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "from and text are required")
		return
	}
	_, _, err := h.inbox.IngestInbound(c.Request.Context(), tenant.ID, services.InboundInput{
		Phone: from, Body: text, Channel: models.ChannelSMS,
		Provider: "africastalking", ProviderMessageID: id,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Evolution inbound WhatsApp.
//
//	@Summary		Evolution WhatsApp webhook
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			tenant_slug	path	string	true	"Tenant slug"
//	@Success		200	{object}	map[string]string
//	@Router			/api/v1/webhooks/{tenant_slug}/whatsapp/evolution [post]
func (h *WebhookHandler) Evolution(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	tenant, ok := h.resolveTenant(c)
	if !ok {
		return
	}

	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid body")
		return
	}

	phone, body, providerID := parseEvolutionPayload(raw)
	if phone == "" || body == "" {
		// Evolution sends many event types; ignore non-message ones.
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}
	_, _, err = h.inbox.IngestInbound(c.Request.Context(), tenant.ID, services.InboundInput{
		Phone: phone, Body: body, Channel: models.ChannelWhatsApp,
		Provider: "evolution", ProviderMessageID: providerID,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func parseEvolutionPayload(raw []byte) (phone, body, id string) {
	var payload struct {
		Event string `json:"event"`
		Data  struct {
			Key struct {
				RemoteJID string `json:"remoteJid"`
				ID        string `json:"id"`
				FromMe    bool   `json:"fromMe"`
			} `json:"key"`
			Message struct {
				Conversation string `json:"conversation"`
				Extended     struct {
					Text string `json:"text"`
				} `json:"extendedTextMessage"`
			} `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", ""
	}
	if payload.Data.Key.FromMe {
		return "", "", ""
	}
	body = payload.Data.Message.Conversation
	if body == "" {
		body = payload.Data.Message.Extended.Text
	}
	phone = strings.Split(payload.Data.Key.RemoteJID, "@")[0]
	return phone, body, payload.Data.Key.ID
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
