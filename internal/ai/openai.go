package ai

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

	"github.com/Osawejustice/nandi-api/internal/config"
)

// OpenAICompatible talks to Groq, OpenAI, or any /chat/completions endpoint.
type OpenAICompatible struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
	log     zerolog.Logger
}

func NewAnalyzer(cfg config.AIConfig, log zerolog.Logger) Analyzer {
	if strings.TrimSpace(cfg.APIKey) == "" {
		log.Warn().Msg("AI_API_KEY empty; sentiment disabled")
		return NoopAnalyzer{}
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.groq.com/openai/v1"
	}
	model := cfg.Model
	if model == "" {
		model = "llama-3.1-8b-instant"
	}
	return &OpenAICompatible{
		apiKey:  cfg.APIKey,
		model:   model,
		baseURL: base,
		client:  &http.Client{Timeout: 12 * time.Second},
		log:     log.With().Str("component", "ai").Logger(),
	}
}

func (a *OpenAICompatible) Analyze(ctx context.Context, text string) (*Result, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return &Result{Score: 0, Label: LabelNeutral}, nil
	}

	prompt := `Classify the customer message sentiment. Reply with JSON only: {"label":"positive|neutral|negative","score":<number from -1 to 1>}. Message: ` + text
	body, err := a.chat(ctx, prompt, false)
	if err != nil {
		return nil, err
	}

	label, score, ok := parseSentimentJSON(body)
	if !ok {
		a.log.Warn().Str("raw", body).Msg("could not parse sentiment json")
		return &Result{Score: 0, Label: LabelNeutral}, nil
	}
	return &Result{Score: score, Label: label}, nil
}

func (a *OpenAICompatible) Summarize(ctx context.Context, thread string) (string, error) {
	prompt := "Summarize this customer support conversation in 3 short sentences for an agent. Mention the issue, sentiment, and any next step.\n\n" + thread
	return a.chat(ctx, prompt, false)
}

func (a *OpenAICompatible) chat(ctx context.Context, userPrompt string, _ bool) (string, error) {
	payload := map[string]any{
		"model": a.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a concise assistant for a customer engagement inbox."},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.1,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ai status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai empty response")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func parseSentimentJSON(raw string) (string, float64, bool) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var parsed struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", 0, false
	}
	label := NormalizeLabel(parsed.Label)
	score := parsed.Score
	if score == 0 {
		score = ScoreFromLabel(label)
	}
	if score > 1 {
		score = 1
	}
	if score < -1 {
		score = -1
	}
	return label, score, true
}

// Summarizer is optionally satisfied by OpenAICompatible.
type Summarizer interface {
	Summarize(ctx context.Context, thread string) (string, error)
}
