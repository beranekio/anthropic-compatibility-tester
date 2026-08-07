package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesThinking verifies POST /v1/messages with extended thinking enabled.
type MessagesThinking struct{}

func (MessagesThinking) Name() string { return "messages_thinking" }
func (MessagesThinking) Description() string {
	return "Message completion with extended thinking (POST /v1/messages)"
}

func (MessagesThinking) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	// BudgetTokens must be ≥1024 and less than max_tokens.
	const maxTokens int64 = 2048
	const budgetTokens int64 = 1024

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("What is 2+2? Reply with one short sentence.")),
		},
		Thinking: anthropic.ThinkingConfigParamOfEnabled(budgetTokens),
	})
	if err != nil {
		return fmt.Errorf("messages thinking request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_thinking", msg); err != nil {
		return err
	}
	if isRefusalStopReason(msg) {
		return nil
	}
	if err := validateMessageHasTextOutput("messages_thinking", msg); err != nil {
		return err
	}
	if !messageHasThinkingContent(msg) {
		return fail("messages_thinking", "response missing thinking or redacted_thinking content block")
	}
	return nil
}

func messageHasThinkingContent(msg *anthropic.Message) bool {
	if msg == nil {
		return false
	}
	for _, block := range msg.Content {
		switch block.Type {
		case "thinking", "redacted_thinking":
			return true
		}
	}
	return false
}
