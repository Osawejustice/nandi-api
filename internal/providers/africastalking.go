package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/yourorg/nandi/internal/config"
	"github.com/yourorg/nandi/internal/models"
)

const (
	atLiveURL    = "https://api.africastalking.com/version1/messaging"
	atSandboxURL = "https://api.sandbox.africastalking.com/version1/messaging"
)

// AfricaTalking implements ChannelProvider for SMS. WhatsApp is not offered here.
type AfricaTalking struct {
	username string
	apiKey   string
	from     string
	endpoint string
	client   *http.Client
	log      zerolog.Logger
}

func NewAfricaTalking(cfg config.AfricaTalkingConfig, log zerolog.Logger) *AfricaTalking {
	endpoint := atLiveURL
	if cfg.Sandbox {
		endpoint = atSandboxURL
	}
	return &AfricaTalking{
		username: cfg.Username,
		apiKey:   cfg.APIKey,
		from:     cfg.SenderID,
		endpoint: endpoint,
		client:   &http.Client{Timeout: 15 * time.Second},
		log:      log.With().Str("provider", "africastalking").Logger(),
	}
}

func (p *AfricaTalking) Name() string { return "africastalking" }

func (p *AfricaTalking) Supports(channel string) bool {
	return channel == models.ChannelSMS
}

func (p *AfricaTalking) Configured() bool {
	return strings.TrimSpace(p.username) != "" && strings.TrimSpace(p.apiKey) != ""
}

func (p *AfricaTalking) SendSMS(ctx context.Context, req SendRequest) (*SendResult, error) {
	if !p.Configured() {
		return nil, ErrNotConfigured
	}

	form := url.Values{}
	form.Set("username", p.username)
	form.Set("to", req.To)
	form.Set("message", req.Body)
	if p.from != "" {
		form.Set("from", p.from)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("apiKey", p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 300 {
		p.log.Error().Int("status", resp.StatusCode).Bytes("body", body).Msg("africastalking send failed")
		return nil, fmt.Errorf("%w: status %d", ErrSendFailed, resp.StatusCode)
	}

	var parsed atResponse
	_ = json.Unmarshal(body, &parsed)
	msgID := parsed.FirstMessageID()
	if msgID == "" {
		msgID = fmt.Sprintf("at-%d", time.Now().UnixNano())
	}

	p.log.Info().Str("to", req.To).Str("provider_message_id", msgID).Msg("africastalking sms accepted")
	return &SendResult{Provider: p.Name(), ProviderMessageID: msgID}, nil
}

func (p *AfricaTalking) SendWhatsApp(_ context.Context, _ SendRequest) (*SendResult, error) {
	return nil, ErrUnsupported
}

type atResponse struct {
	SMSMessageData struct {
		Recipients []struct {
			MessageID string `json:"messageId"`
			Status    string `json:"status"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

func (r atResponse) FirstMessageID() string {
	if len(r.SMSMessageData.Recipients) == 0 {
		return ""
	}
	return r.SMSMessageData.Recipients[0].MessageID
}

var _ ChannelProvider = (*AfricaTalking)(nil)
