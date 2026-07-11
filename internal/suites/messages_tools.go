package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesTools verifies tool use on POST /v1/messages.
type MessagesTools struct{}

func (MessagesTools) Name() string { return "messages_tools" }
func (MessagesTools) Description() string {
	return "Message completion with tools (POST /v1/messages, tool_choice any)"
}

func (MessagesTools) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:      anthropic.Model(cfg.Model),
		MaxTokens:  256,
		Messages:   []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("What is the weather in San Francisco?"))},
		Tools:      weatherTools(),
		ToolChoice: requiredToolChoice(),
	})
	if err != nil {
		return fmt.Errorf("messages tools request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_tools", msg); err != nil {
		return err
	}
	if isRefusalStopReason(msg) {
		return nil
	}
	if !messageHasToolUse(msg) {
		return fail("messages_tools", "response has no tool_use content")
	}
	if !isToolUseStopReason(msg) {
		return fail("messages_tools", fmt.Sprintf("stop_reason is %q, want tool_use", msg.StopReason))
	}
	for _, block := range messageToolUseBlocks(msg) {
		if err := validateToolUseBlock("messages_tools", block); err != nil {
			return err
		}
	}
	return nil
}