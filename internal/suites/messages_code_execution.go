package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesCodeExecution verifies POST /v1/messages with the server code_execution tool.
type MessagesCodeExecution struct{}

func (MessagesCodeExecution) Name() string { return "messages_code_execution" }
func (MessagesCodeExecution) Description() string {
	return "Message completion with server code_execution tool (POST /v1/messages)"
}

func (MessagesCodeExecution) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Use the code execution tool to compute 1+1.")),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfCodeExecutionTool20250522: &anthropic.CodeExecutionTool20250522Param{},
		}},
		// Force a tool path; auto tool_choice can answer with plain text and never invoke.
		ToolChoice: requiredToolChoice(),
	})
	if err != nil {
		return fmt.Errorf("messages code execution request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_code_execution", msg); err != nil {
		return err
	}
	if isRefusalStopReason(msg) {
		return nil
	}
	if messageHasServerToolUse(msg, "code_execution") {
		return nil
	}
	// Some gateways may surface server tools as ordinary tool_use.
	if messageHasNamedToolUse(msg, "code_execution") {
		return nil
	}
	return fail("messages_code_execution", "response missing server_tool_use or tool_use for code_execution")
}

func messageHasServerToolUse(msg *anthropic.Message, wantName string) bool {
	if msg == nil {
		return false
	}
	for _, block := range msg.Content {
		if block.Type == "server_tool_use" {
			if wantName == "" || block.Name == wantName {
				return true
			}
		}
	}
	return false
}

func messageHasNamedToolUse(msg *anthropic.Message, wantName string) bool {
	if msg == nil {
		return false
	}
	for _, block := range msg.Content {
		if block.Type == "tool_use" && block.Name == wantName {
			return true
		}
	}
	return false
}
