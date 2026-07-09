package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesCountTokens verifies POST /v1/messages/count_tokens.
type MessagesCountTokens struct{}

func (MessagesCountTokens) Name() string { return "messages_count_tokens" }
func (MessagesCountTokens) Description() string {
	return "Token counting (POST /v1/messages/count_tokens)"
}

func (MessagesCountTokens) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	count, err := client.Messages.CountTokens(ctx, anthropic.MessageCountTokensParams{
		Model: anthropic.Model(cfg.Model),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with exactly the word: pong")),
		},
	})
	if err != nil {
		return fmt.Errorf("messages count_tokens request failed: %w", err)
	}
	if count == nil {
		return fail("messages_count_tokens", "response is nil")
	}
	if count.InputTokens <= 0 {
		return fail("messages_count_tokens", "input_tokens must be greater than zero")
	}
	return nil
}