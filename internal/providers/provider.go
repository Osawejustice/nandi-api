package providers

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Osawejustice/nandi-api/internal/models"
)

var (
	ErrNotConfigured = errors.New("provider not configured")
	ErrUnsupported   = errors.New("channel not supported by provider")
	ErrSendFailed    = errors.New("provider send failed")
)

// SendRequest is the provider-agnostic outbound payload.
type SendRequest struct {
	TenantID  uuid.UUID
	To        string
	Body      string
	Channel   string
	MessageID uuid.UUID
	Metadata  map[string]string
}

// SendResult is what a successful adapter returns.
type SendResult struct {
	Provider          string
	ProviderMessageID string
}

// ChannelProvider is the only type core services may depend on.
type ChannelProvider interface {
	Name() string
	Supports(channel string) bool
	SendSMS(ctx context.Context, req SendRequest) (*SendResult, error)
	SendWhatsApp(ctx context.Context, req SendRequest) (*SendResult, error)
}

func Send(ctx context.Context, p ChannelProvider, req SendRequest) (*SendResult, error) {
	switch req.Channel {
	case models.ChannelSMS:
		return p.SendSMS(ctx, req)
	case models.ChannelWhatsApp:
		return p.SendWhatsApp(ctx, req)
	default:
		return nil, ErrUnsupported
	}
}
