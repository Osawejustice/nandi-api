package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/Osawejustice/nandi-api/internal/config"
	"github.com/Osawejustice/nandi-api/internal/models"
	"github.com/Osawejustice/nandi-api/internal/repositories"
)

// Router tries providers in configured order and writes a ProviderLog per attempt.
// Core services depend on this, never on a concrete adapter.
type Router struct {
	sms      []ChannelProvider
	whatsapp []ChannelProvider
	logs     *repositories.ProviderLogRepo
	log      zerolog.Logger
}

func NewRouter(cfg config.ProviderConfig, registry map[string]ChannelProvider, logs *repositories.ProviderLogRepo, log zerolog.Logger) *Router {
	return &Router{
		sms:      resolveChain(cfg.PrimarySMS, cfg.FailoverSMS, registry),
		whatsapp: resolveChain(cfg.PrimaryWhatsApp, cfg.FailoverWhatsApp, registry),
		logs:     logs,
		log:      log.With().Str("component", "provider_router").Logger(),
	}
}

func resolveChain(primary, failover string, registry map[string]ChannelProvider) []ChannelProvider {
	var out []ChannelProvider
	seen := map[string]struct{}{}
	for _, name := range []string{strings.ToLower(strings.TrimSpace(primary)), strings.ToLower(strings.TrimSpace(failover))} {
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		if p, ok := registry[name]; ok {
			out = append(out, p)
			seen[name] = struct{}{}
		}
	}
	return out
}

func (r *Router) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	chain := r.sms
	if req.Channel == models.ChannelWhatsApp {
		chain = r.whatsapp
	}
	if len(chain) == 0 {
		return nil, fmt.Errorf("%w: no providers configured for %s", ErrUnsupported, req.Channel)
	}

	var last error
	attempt := 0
	for _, p := range chain {
		if !p.Supports(req.Channel) {
			continue
		}
		attempt++
		start := time.Now()
		result, err := Send(ctx, p, req)
		r.record(ctx, req, p.Name(), attempt, result, err, time.Since(start))
		if err == nil && result != nil {
			return result, nil
		}
		last = err
		r.log.Warn().Err(err).Str("provider", p.Name()).Str("channel", req.Channel).Int("attempt", attempt).Msg("provider failed; trying next")
	}
	if last == nil {
		last = ErrSendFailed
	}
	return nil, last
}

func (r *Router) record(ctx context.Context, req SendRequest, provider string, attempt int, result *SendResult, err error, latency time.Duration) {
	if r.logs == nil {
		return
	}
	status := "success"
	if err != nil {
		status = "failed"
	}
	row := &models.ProviderLog{
		ID:        models.NewID(),
		TenantID:  req.TenantID,
		Provider:  provider,
		Channel:   req.Channel,
		To:        req.To,
		Attempt:   attempt,
		Status:    status,
		Success:   err == nil,
		LatencyMS: latency.Milliseconds(),
		CreatedAt: time.Now().UTC(),
	}
	if req.MessageID != uuid.Nil {
		id := req.MessageID
		row.MessageID = &id
	}
	if result != nil {
		row.ProviderMessageID = result.ProviderMessageID
	}
	if err != nil {
		row.ErrorMessage = err.Error()
	}
	if logErr := r.logs.Create(ctx, row); logErr != nil {
		r.log.Error().Err(logErr).Msg("write provider log")
	}
}

func (r *Router) Status() map[string][]string {
	names := func(list []ChannelProvider) []string {
		out := make([]string, 0, len(list))
		for _, p := range list {
			out = append(out, p.Name())
		}
		return out
	}
	return map[string][]string{
		models.ChannelSMS:      names(r.sms),
		models.ChannelWhatsApp: names(r.whatsapp),
	}
}
