package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/yourorg/nandi/internal/config"
	"github.com/yourorg/nandi/internal/models"
)

// Evolution talks to Evolution API / Evolution Go over REST for WhatsApp.
type Evolution struct {
	baseURL  string
	apiKey   string
	instance string
	client   *http.Client
	log      zerolog.Logger
}

func NewEvolution(cfg config.EvolutionConfig, log zerolog.Logger) *Evolution {
	return &Evolution{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:   cfg.APIKey,
		instance: cfg.Instance,
		client:   &http.Client{Timeout: 20 * time.Second},
		log:      log.With().Str("provider", "evolution").Logger(),
	}
}

func (p *Evolution) Name() string { return "evolution" }

func (p *Evolution) Supports(channel string) bool {
	return channel == models.ChannelWhatsApp
}

func (p *Evolution) Configured() bool {
	return p.baseURL != "" && p.apiKey != "" && p.instance != ""
}

func (p *Evolution) SendSMS(_ context.Context, _ SendRequest) (*SendResult, error) {
	return nil, ErrUnsupported
}

func (p *Evolution) SendWhatsApp(ctx context.Context, req SendRequest) (*SendResult, error) {
	if !p.Configured() {
		return nil, ErrNotConfigured
	}

	payload := map[string]any{
		"number": req.To,
		"text":   req.Body,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/message/sendText/%s", p.baseURL, p.instance)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 300 {
		p.log.Error().Int("status", resp.StatusCode).Bytes("body", body).Msg("evolution send failed")
		return nil, fmt.Errorf("%w: status %d", ErrSendFailed, resp.StatusCode)
	}

	var parsed struct {
		Key struct {
			ID string `json:"id"`
		} `json:"key"`
	}
	_ = json.Unmarshal(body, &parsed)
	msgID := parsed.Key.ID
	if msgID == "" {
		msgID = fmt.Sprintf("evo-%d", time.Now().UnixNano())
	}

	p.log.Info().Str("to", req.To).Str("provider_message_id", msgID).Msg("evolution whatsapp accepted")
	return &SendResult{Provider: p.Name(), ProviderMessageID: msgID}, nil
}

var _ ChannelProvider = (*Evolution)(nil)
