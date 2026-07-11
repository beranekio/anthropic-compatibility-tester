package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// Completions verifies legacy POST /v1/complete via client.Completions.New.
type Completions struct{}

func (Completions) Name() string { return "completions" }
func (Completions) Description() string {
	return "(deprecated) Text completion (POST /v1/complete)"
}

func (Completions) Deprecated() bool { return true }

func (Completions) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	resp, err := client.Completions.New(ctx, anthropic.CompletionNewParams{
		Model:             anthropic.Model(cfg.CompletionModel),
		MaxTokensToSample: 64,
		Prompt:            "\n\nHuman: Reply with exactly the word: pong\n\nAssistant:",
	})
	if err != nil {
		return fmt.Errorf("completions request failed: %w", err)
	}
	return validateCompletionEnvelope("completions", resp)
}