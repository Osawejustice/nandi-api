package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/yourorg/nandi/internal/config"
	"github.com/yourorg/nandi/internal/models"
)

type failProvider struct{}

func (failProvider) Name() string                 { return "fail" }
func (failProvider) Supports(channel string) bool { return channel == models.ChannelSMS }
func (failProvider) SendSMS(context.Context, SendRequest) (*SendResult, error) {
	return nil, errors.New("primary down")
}
func (failProvider) SendWhatsApp(context.Context, SendRequest) (*SendResult, error) {
	return nil, ErrUnsupported
}

func TestRouterFailoversToStub(t *testing.T) {
	log := zerolog.Nop()
	registry := map[string]ChannelProvider{
		"fail": NewFailForTest(),
		"stub": NewStubProvider(log),
	}
	r := NewRouter(config.ProviderConfig{PrimarySMS: "fail", FailoverSMS: "stub"}, registry, nil, log)
	res, err := r.Send(context.Background(), SendRequest{
		TenantID: uuid.New(),
		To:       "+254700000001",
		Body:     "hello",
		Channel:  models.ChannelSMS,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Provider != "stub" {
		t.Fatalf("expected stub, got %s", res.Provider)
	}
}

func NewFailForTest() ChannelProvider { return failProvider{} }
