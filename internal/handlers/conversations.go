package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yourorg/nandi/internal/middleware"
	"github.com/yourorg/nandi/internal/models"
	"github.com/yourorg/nandi/internal/services"
)

type ConversationHandler struct {
	inbox *services.InboxService
}

func NewConversationHandler(inbox *services.InboxService) *ConversationHandler {
	return &ConversationHandler{inbox: inbox}
}

type ReplyRequest struct {
	Body string `json:"body" binding:"required" example:"Thanks for reaching out."`
}

type PatchConversationRequest struct {
	Status     *string `json:"status" example:"resolved"`
	AssigneeID *string `json:"assignee_id"`
}

type ConversationDetail struct {
	Conversation models.Conversation `json:"conversation"`
	Messages     []models.Message    `json:"messages"`
}

type SummaryData struct {
	Summary string `json:"summary"`
}

// List inbox conversations.
//
//	@Summary		List conversations
//	@Tags			conversations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query	string	false	"open|pending|resolved|closed"
//	@Param			channel		query	string	false	"sms|whatsapp"
//	@Param			assignee_id	query	string	false	"user id or unassigned"
//	@Param			q			query	string	false	"Search"
//	@Param			page		query	int		false	"Page"
//	@Param			per_page	query	int		false	"Page size"
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/conversations [get]
func (h *ConversationHandler) List(c *gin.Context) {
	page, perPage := pageParams(c)
	items, total, err := h.inbox.ListConversations(c.Request.Context(), middleware.TenantID(c), services.ListConversationsInput{
		Status: c.Query("status"), Channel: c.Query("channel"), AssigneeID: c.Query("assignee_id"),
		Query: c.Query("q"), Page: page, PerPage: perPage,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writePage(c, conversationDTOs(items), total, page, perPage)
}

// Get conversation + messages.
//
//	@Summary		Get conversation
//	@Tags			conversations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Conversation ID"
//	@Success		200	{object}	map[string]ConversationDetailDTO
//	@Router			/api/v1/conversations/{id} [get]
func (h *ConversationHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	conv, msgs, err := h.inbox.GetConversation(c.Request.Context(), middleware.TenantID(c), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	_ = h.inbox.MarkRead(c.Request.Context(), middleware.TenantID(c), id)
	writeData(c, http.StatusOK, ConversationDetailDTO{
		Conversation: conversationDTO(conv),
		Messages:     messageDTOs(msgs),
	})
}

// Reply sends an outbound message through ChannelProvider.
//
//	@Summary		Reply in conversation
//	@Tags			conversations
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string			true	"Conversation ID"
//	@Param			body	body	ReplyRequest	true	"Message"
//	@Success		201	{object}	map[string]MessageDTO
//	@Router			/api/v1/conversations/{id}/messages [post]
func (h *ConversationHandler) Reply(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	var req ReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	var actor *uuid.UUID
	if middleware.HasUser(c) {
		uid := middleware.UserID(c)
		actor = &uid
	}
	msg, err := h.inbox.Reply(c.Request.Context(), middleware.TenantID(c), actor, id, req.Body)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusCreated, messageDTO(*msg))
}

// Patch conversation status / assignee.
//
//	@Summary		Update conversation
//	@Tags			conversations
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string						true	"Conversation ID"
//	@Param			body	body	PatchConversationRequest	true	"Patch"
//	@Success		200	{object}	map[string]ConversationDTO
//	@Router			/api/v1/conversations/{id} [patch]
func (h *ConversationHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	var req PatchConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	conv, err := h.inbox.UpdateConversation(c.Request.Context(), middleware.TenantID(c), id, services.UpdateConversationInput{
		Status: req.Status, AssigneeID: req.AssigneeID,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, conversationDTO(conv))
}

// Summarize a thread via LLM.
//
//	@Summary		Summarize conversation
//	@Tags			conversations
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Conversation ID"
//	@Success		200	{object}	map[string]SummaryData
//	@Router			/api/v1/conversations/{id}/summary [post]
func (h *ConversationHandler) Summarize(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	summary, err := h.inbox.Summarize(c.Request.Context(), middleware.TenantID(c), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, SummaryData{Summary: summary})
}

type SimulateInboundRequest struct {
	Phone   string `json:"phone" binding:"required" example:"+254700000001"`
	Name    string `json:"name" example:"Customer"`
	Body    string `json:"body" binding:"required" example:"Hello, I need help"`
	Channel string `json:"channel" example:"sms"`
}

// Simulate inbound (demo / sandbox).
//
//	@Summary		Simulate inbound message
//	@Tags			dev
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	SimulateInboundRequest	true	"Inbound"
//	@Success		201	{object}	map[string]any
//	@Router			/api/v1/dev/inbound [post]
func (h *ConversationHandler) SimulateInbound(c *gin.Context) {
	var req SimulateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	conv, msg, err := h.inbox.IngestInbound(c.Request.Context(), middleware.TenantID(c), services.InboundInput{
		Phone: req.Phone, Name: req.Name, Body: req.Body, Channel: req.Channel, Provider: "simulate",
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusCreated, gin.H{"conversation": conversationDTO(conv), "message": messageDTO(*msg)})
}
