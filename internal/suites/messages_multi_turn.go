package suites

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

const (
	multiTurnToolUseID        = "toolu_mock_weather"
	multiTurnToolResultJSON   = `{"temperature": 72, "unit": "fahrenheit", "condition": "sunny"}`
	multiTurnExpectedTempF    = "72"
)

// MessagesMultiTurn verifies multi-turn POST /v1/messages with tool results.
type MessagesMultiTurn struct{}

func (MessagesMultiTurn) Name() string { return "messages_multi_turn" }
func (MessagesMultiTurn) Description() string {
	return "Multi-turn message completion with tool result history (POST /v1/messages)"
}

func (MessagesMultiTurn) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 256,
		System: []anthropic.TextBlockParam{
			{Text: "You are a helpful assistant. Use the weather tool result when answering follow-up questions."},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("What is the weather in San Francisco?")),
			multiTurnAssistantToolUseMessage(),
			anthropic.NewUserMessage(anthropic.NewToolResultBlock(multiTurnToolUseID, multiTurnToolResultJSON, false)),
			anthropic.NewUserMessage(anthropic.NewTextBlock("What temperature did the weather tool report in Fahrenheit? Reply with the number only.")),
		},
		Tools:      weatherTools(),
		ToolChoice: noneToolChoice(),
	})
	if err != nil {
		return fmt.Errorf("multi-turn messages request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_multi_turn", msg); err != nil {
		return err
	}
	if isRefusalStopReason(msg) {
		return nil
	}
	text := messageTextContent(msg)
	if !strings.Contains(text, multiTurnExpectedTempF) {
		return fail("messages_multi_turn", fmt.Sprintf("response text is %q, want response containing %q from tool context", text, multiTurnExpectedTempF))
	}
	return nil
}

func multiTurnAssistantToolUseMessage() anthropic.MessageParam {
	return anthropic.NewAssistantMessage(anthropic.ContentBlockParamUnion{
		OfToolUse: &anthropic.ToolUseBlockParam{
			ID:    multiTurnToolUseID,
			Name:  weatherToolName,
			Input: map[string]any{"location": "San Francisco, CA"},
		},
	})
}