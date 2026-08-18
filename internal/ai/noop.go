package ai

import "context"

// NoopAnalyzer is used when no AI key is configured. Inbox never blocks on AI.
type NoopAnalyzer struct{}

func (NoopAnalyzer) Analyze(_ context.Context, _ string) (*Result, error) {
	return nil, nil
}
