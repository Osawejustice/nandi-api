package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Osawejustice/nandi-api/internal/middleware"
	"github.com/Osawejustice/nandi-api/internal/models"
	"github.com/Osawejustice/nandi-api/internal/services"
)

type ContactHandler struct {
	svc *services.ContactService
}

func NewContactHandler(svc *services.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
}

type ContactRequest struct {
	Name     string         `json:"name" binding:"required" example:"Ada Lovelace"`
	Phone    string         `json:"phone" binding:"required" example:"+254700000001"`
	Email    string         `json:"email" example:"ada@example.com"`
	Tags     []string       `json:"tags"`
	Metadata models.JSONMap `json:"metadata"`
}

type PatchContactRequest struct {
	Name     string         `json:"name"`
	Phone    string         `json:"phone"`
	Email    string         `json:"email"`
	Tags     []string       `json:"tags"`
	Metadata models.JSONMap `json:"metadata"`
}

// List contacts.
//
//	@Summary		List contacts
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q		query	string	false	"Search"
//	@Param			tag		query	string	false	"Tag filter"
//	@Param			page	query	int		false	"Page"
//	@Param			per_page	query	int	false	"Page size"
//	@Success		200	{object}	map[string]any
//	@Router			/api/v1/contacts [get]
func (h *ContactHandler) List(c *gin.Context) {
	page, perPage := pageParams(c)
	items, total, err := h.svc.List(c.Request.Context(), middleware.TenantID(c), c.Query("q"), c.Query("tag"), page, perPage)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writePage(c, contactDTOs(items), total, page, perPage)
}

// Create a contact.
//
//	@Summary		Create contact
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	ContactRequest	true	"Contact"
//	@Success		201	{object}	map[string]ContactDTO
//	@Router			/api/v1/contacts [post]
func (h *ContactHandler) Create(c *gin.Context) {
	var req ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	item, err := h.svc.Create(c.Request.Context(), middleware.TenantID(c), services.ContactInput{
		Name: req.Name, Phone: req.Phone, Email: req.Email, Tags: req.Tags, Metadata: req.Metadata,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusCreated, contactDTO(item))
}

// Get a contact.
//
//	@Summary		Get contact
//	@Tags			contacts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Contact ID"
//	@Success		200	{object}	map[string]ContactDTO
//	@Router			/api/v1/contacts/{id} [get]
func (h *ContactHandler) Get(c *gin.Context) {
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
	writeData(c, http.StatusOK, contactDTO(item))
}

// Patch a contact.
//
//	@Summary		Update contact
//	@Tags			contacts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string				true	"Contact ID"
//	@Param			body	body	PatchContactRequest	true	"Contact"
//	@Success		200	{object}	map[string]ContactDTO
//	@Router			/api/v1/contacts/{id} [patch]
func (h *ContactHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	var req PatchContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	item, err := h.svc.Update(c.Request.Context(), middleware.TenantID(c), id, services.ContactInput{
		Name: req.Name, Phone: req.Phone, Email: req.Email, Tags: req.Tags, Metadata: req.Metadata,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusOK, contactDTO(item))
}

// Delete a contact (soft).
//
//	@Summary		Delete contact
//	@Tags			contacts
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Contact ID"
//	@Success		204
//	@Router			/api/v1/contacts/{id} [delete]
func (h *ContactHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), middleware.TenantID(c), id); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
