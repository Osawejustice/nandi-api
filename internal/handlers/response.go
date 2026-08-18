package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/Osawejustice/nandi-api/internal/middleware"
	"github.com/Osawejustice/nandi-api/internal/services"
)

// ErrorBody is the consistent API error envelope.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string            `json:"code" example:"validation_error"`
	Message   string            `json:"message" example:"invalid request"`
	RequestID string            `json:"request_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Details   map[string]string `json:"details,omitempty"`
	Tenants   any               `json:"tenants,omitempty"`
}

type DataBody struct {
	Data any `json:"data"`
}

func writeData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: ErrorDetail{
		Code:      code,
		Message:   message,
		RequestID: middleware.GetRequestID(c),
	}})
}

func writeErrorDetails(c *gin.Context, status int, code, message string, details map[string]string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: ErrorDetail{
		Code:      code,
		Message:   message,
		RequestID: middleware.GetRequestID(c),
		Details:   details,
	}})
}

func handleBindError(c *gin.Context, err error) {
	details := map[string]string{}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		for _, fe := range verrs {
			details[fe.Field()] = fe.Tag()
		}
	}
	writeErrorDetails(c, http.StatusBadRequest, "validation_error", "invalid request", details)
}

func handleServiceError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if mt, ok := services.IsMultiTenant(err); ok {
		c.AbortWithStatusJSON(http.StatusConflict, ErrorBody{Error: ErrorDetail{
			Code:      "tenant_required",
			Message:   mt.Error(),
			RequestID: middleware.GetRequestID(c),
			Tenants:   mt.Tenants,
		}})
		return
	}

	switch {
	case errors.Is(err, services.ErrValidation):
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, services.ErrInvalidCredentials):
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, services.ErrUnauthorized), errors.Is(err, services.ErrInvalidToken):
		writeError(c, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, services.ErrForbidden):
		writeError(c, http.StatusForbidden, "forbidden", "insufficient permissions")
	case errors.Is(err, services.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, services.ErrEmailTaken):
		writeError(c, http.StatusConflict, "email_taken", "email already registered for this organization")
	case errors.Is(err, services.ErrSlugTaken):
		writeError(c, http.StatusConflict, "slug_taken", "organization slug already taken")
	case errors.Is(err, services.ErrTenantSuspended):
		writeError(c, http.StatusForbidden, "tenant_suspended", "organization is suspended")
	case errors.Is(err, services.ErrTenantRequired):
		writeError(c, http.StatusConflict, "tenant_required", err.Error())
	case errors.Is(err, services.ErrUnavailable):
		writeError(c, http.StatusServiceUnavailable, "unavailable", "database is not available")
	case errors.Is(err, services.ErrUnsupportedChannel):
		writeError(c, http.StatusBadRequest, "unsupported_channel", err.Error())
	case errors.Is(err, services.ErrInvalidState):
		writeError(c, http.StatusConflict, "invalid_state", err.Error())
	case errors.Is(err, services.ErrProvider):
		writeError(c, http.StatusBadGateway, "provider_error", "message could not be sent")
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
