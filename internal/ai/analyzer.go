package ai

import (
	"context"
	"strings"
)

const (
	LabelPositive = "positive"
	LabelNeutral  = "neutral"
	LabelNegative = "negative"
)

// Result is stored on Message and rolled up onto Conversation.
type Result struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

// Analyzer scores inbound text. Implementations must be safe to call in a goroutine.
type Analyzer interface {
	Analyze(ctx context.Context, text string) (*Result, error)
}

func NormalizeLabel(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case LabelPositive, "pos":
		return LabelPositive
	case LabelNegative, "neg":
		return LabelNegative
	default:
		return LabelNeutral
	}
}

func ScoreFromLabel(label string) float64 {
	switch NormalizeLabel(label) {
	case LabelPositive:
		return 0.75
	case LabelNegative:
		return -0.75
	default:
		return 0
	}
}
