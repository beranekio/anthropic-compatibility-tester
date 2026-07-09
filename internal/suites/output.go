package suites

import (
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

func validateMessageEnvelope(suite string, msg *anthropic.Message) error {
	if msg == nil {
		return fail(suite, "response is nil")
	}
	if msg.ID == "" {
		return fail(suite, "response missing id")
	}
	if msg.Model == "" {
		return fail(suite, "response missing model")
	}
	if string(msg.Type) != "message" {
		return fail(suite, fmt.Sprintf("response type is %q, want message", msg.Type))
	}
	if string(msg.Role) != "assistant" {
		return fail(suite, fmt.Sprintf("response role is %q, want assistant", msg.Role))
	}
	if msg.StopReason == "" {
		return fail(suite, "response missing stop_reason")
	}
	return nil
}

func messageTextContent(msg *anthropic.Message) string {
	if msg == nil {
		return ""
	}
	var b strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" && block.Text != "" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}

func messageHasToolUse(msg *anthropic.Message) bool {
	if msg == nil {
		return false
	}
	for _, block := range msg.Content {
		if block.Type == "tool_use" {
			return true
		}
	}
	return false
}

func messageToolUseBlocks(msg *anthropic.Message) []anthropic.ContentBlockUnion {
	if msg == nil {
		return nil
	}
	var blocks []anthropic.ContentBlockUnion
	for _, block := range msg.Content {
		if block.Type == "tool_use" {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func validateMessageHasTextOutput(suite string, msg *anthropic.Message) error {
	if isRefusalStopReason(msg) {
		return nil
	}
	if messageTextContent(msg) != "" {
		return nil
	}
	if messageHasToolUse(msg) {
		return fail(suite, "response has tool_use but no tools were requested")
	}
	return fail(suite, "response produced no text content")
}

func isRefusalStopReason(msg *anthropic.Message) bool {
	if msg == nil {
		return false
	}
	return string(msg.StopReason) == "refusal"
}

func isToolUseStopReason(msg *anthropic.Message) bool {
	if msg == nil {
		return false
	}
	return string(msg.StopReason) == "tool_use"
}

func validateToolUseBlock(suite string, block anthropic.ContentBlockUnion) error {
	if block.ID == "" {
		return fail(suite, "tool_use block missing id")
	}
	if block.Name == "" {
		return fail(suite, "tool_use block missing name")
	}
	if block.Name != weatherToolName {
		return fail(suite, fmt.Sprintf("tool_use name is %q, want %s", block.Name, weatherToolName))
	}
	return validateWeatherToolInput(suite, block.Input)
}