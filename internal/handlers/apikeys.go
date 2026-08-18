package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Osawejustice/nandi-api/internal/middleware"
	"github.com/Osawejustice/nandi-api/internal/services"
)

type APIKeyHandler struct {
	auth *services.AuthService
}

func NewAPIKeyHandler(auth *services.AuthService) *APIKeyHandler {
	return &APIKeyHandler{auth: auth}
}

// Create issues a new API key. The raw key is returned once.
//
//	@Summary		Create API key
//	@Tags			api-keys
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateAPIKeyRequest	true	"Key"
//	@Success		201		{object}	map[string]APIKeyCreatedData
//	@Failure		400		{object}	ErrorBody
//	@Router			/api/v1/api-keys [post]
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}
	key, raw, err := h.auth.CreateAPIKey(c.Request.Context(), middleware.TenantID(c), middleware.UserID(c), req.Name, req.Role)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	writeData(c, http.StatusCreated, APIKeyCreatedData{APIKeyDTO: apiKeyDTO(*key), Key: raw})
}

// List returns tenant API keys (never the raw secret).
//
//	@Summary		List API keys
//	@Tags			api-keys
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string][]APIKeyDTO
//	@Router			/api/v1/api-keys [get]
func (h *APIKeyHandler) List(c *gin.Context) {
	keys, err := h.auth.ListAPIKeys(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	out := make([]APIKeyDTO, 0, len(keys))
	for _, k := range keys {
		out = append(out, apiKeyDTO(k))
	}
	writeData(c, http.StatusOK, out)
}

// Revoke soft-deletes an API key.
//
//	@Summary		Revoke API key
//	@Tags			api-keys
//	@Security		BearerAuth
//	@Param			id	path	string	true	"API key ID"
//	@Success		204
//	@Failure		404	{object}	ErrorBody
//	@Router			/api/v1/api-keys/{id} [delete]
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid id")
		return
	}
	if err := h.auth.RevokeAPIKey(c.Request.Context(), middleware.TenantID(c), id); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
