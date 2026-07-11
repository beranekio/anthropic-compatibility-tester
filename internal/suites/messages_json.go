package suites

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesJSON verifies POST /v1/messages with structured JSON output.
type MessagesJSON struct{}

func (MessagesJSON) Name() string { return "messages_json" }
func (MessagesJSON) Description() string {
	return "Message structured JSON output (POST /v1/messages, output_format json_schema)"
}

func (MessagesJSON) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with JSON containing an answer field")),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"answer": map[string]any{"type": "string"},
					},
					"required":             []string{"answer"},
					"additionalProperties": false,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("messages json request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_json", msg); err != nil {
		return err
	}
	if isRefusalStopReason(msg) {
		return nil
	}
	text := messageTextContent(msg)
	if text == "" {
		return fail("messages_json", "response has no text content")
	}
	return validateStructuredAnswerJSON("messages_json", text)
}

func validateStructuredAnswerJSON(suite string, content string) error {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return fail(suite, fmt.Sprintf("message content is not valid JSON: %v", err))
	}
	if len(parsed) != 1 {
		return fail(suite, fmt.Sprintf("parsed JSON has %d top-level fields, want 1", len(parsed)))
	}
	answer, ok := parsed["answer"]
	if !ok {
		return fail(suite, `parsed JSON missing "answer" field`)
	}
	if _, ok := answer.(string); !ok {
		return fail(suite, `"answer" field is not a string`)
	}
	return nil
}