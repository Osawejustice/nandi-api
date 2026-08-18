package providers

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/Osawejustice/nandi-api/internal/models"
)

// StubProvider is the local/demo failover. It never talks to a network.
type StubProvider struct {
	log zerolog.Logger
}

func NewStubProvider(log zerolog.Logger) *StubProvider {
	return &StubProvider{log: log.With().Str("provider", "stub").Logger()}
}

func (p *StubProvider) Name() string { return "stub" }

func (p *StubProvider) Supports(channel string) bool {
	return channel == models.ChannelSMS || channel == models.ChannelWhatsApp
}

func (p *StubProvider) SendSMS(ctx context.Context, req SendRequest) (*SendResult, error) {
	return p.send(req, models.ChannelSMS)
}

func (p *StubProvider) SendWhatsApp(ctx context.Context, req SendRequest) (*SendResult, error) {
	return p.send(req, models.ChannelWhatsApp)
}

func (p *StubProvider) send(req SendRequest, channel string) (*SendResult, error) {
	id := "stub-" + uuid.NewString()
	p.log.Warn().
		Str("to", req.To).
		Str("channel", channel).
		Str("provider_message_id", id).
		Msg("stub provider accepted outbound message")
	return &SendResult{Provider: p.Name(), ProviderMessageID: id}, nil
}

var _ ChannelProvider = (*StubProvider)(nil)
