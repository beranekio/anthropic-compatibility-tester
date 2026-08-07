package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesPromptCache verifies POST /v1/messages with cache_control breakpoints.
type MessagesPromptCache struct{}

func (MessagesPromptCache) Name() string { return "messages_prompt_cache" }
func (MessagesPromptCache) Description() string {
	return "Message completion with prompt cache_control (POST /v1/messages)"
}

func (MessagesPromptCache) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	systemBlock := anthropic.TextBlockParam{
		Text:         "You are a concise assistant used only for compatibility testing.",
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
	}
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 64,
		System:    []anthropic.TextBlockParam{systemBlock},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with exactly the word: pong")),
		},
	})
	if err != nil {
		return fmt.Errorf("messages prompt cache request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_prompt_cache", msg); err != nil {
		return err
	}
	// Cache usage fields are optional across providers; only require a valid message envelope/output.
	return validateMessageHasTextOutput("messages_prompt_cache", msg)
}
