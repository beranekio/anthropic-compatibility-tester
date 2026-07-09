package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// Messages verifies POST /v1/messages via client.Messages.New.
type Messages struct{}

func (Messages) Name() string { return "messages" }
func (Messages) Description() string {
	return "Message completion (POST /v1/messages)"
}

func (Messages) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 64,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with exactly the word: pong")),
		},
	})
	if err != nil {
		return fmt.Errorf("messages request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages", msg); err != nil {
		return err
	}
	return validateMessageHasOutput("messages", msg)
}