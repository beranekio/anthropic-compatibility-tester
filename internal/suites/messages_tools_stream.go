package suites

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesToolsStream verifies streaming tool use on POST /v1/messages.
type MessagesToolsStream struct{}

func (MessagesToolsStream) Name() string { return "messages_tools_stream" }
func (MessagesToolsStream) Description() string {
	return "Streaming message completion with tools (POST /v1/messages, stream=true)"
}

func (MessagesToolsStream) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var httpResp *http.Response
	stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:      anthropic.Model(cfg.Model),
		MaxTokens:  256,
		Messages:   []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("What is the weather in San Francisco?"))},
		Tools:      weatherTools(),
		ToolChoice: requiredToolChoice(),
	}, option.WithResponseInto(&httpResp))
	defer stream.Close()

	if err := stream.Err(); err != nil {
		return fmt.Errorf("messages tools stream failed: %w", err)
	}
	if err := validateEventStreamContentType("messages_tools_stream", httpResp); err != nil {
		return err
	}

	var hasToolUse bool
	var finished bool
	var stopReason string
	var toolUseID, toolUseName string
	var toolInput strings.Builder
	var inToolUse bool
	for stream.Next() {
		event := stream.Current()
		switch event.Type {
		case "content_block_start":
			block := event.AsContentBlockStart().ContentBlock
			if block.Type == "tool_use" {
				inToolUse = true
				toolUseID = block.ID
				toolUseName = block.Name
			}
		case "content_block_delta":
			delta := event.AsContentBlockDelta().Delta
			if inToolUse && delta.Type == "input_json_delta" {
				toolInput.WriteString(delta.PartialJSON)
			}
		case "content_block_stop":
			if inToolUse {
				if err := validateStreamedToolUse("messages_tools_stream", toolUseID, toolUseName, toolInput.String()); err != nil {
					return err
				}
				hasToolUse = true
				inToolUse = false
			}
		case "message_stop":
			finished = true
		case "message_delta":
			if event.AsMessageDelta().Delta.StopReason != "" {
				stopReason = string(event.AsMessageDelta().Delta.StopReason)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("messages tools stream failed: %w", err)
	}
	if !finished {
		return fail("messages_tools_stream", "stream missing terminal message_stop event")
	}
	if stopReason == "refusal" {
		return nil
	}
	if !hasToolUse {
		return fail("messages_tools_stream", "stream produced no tool_use content block")
	}
	if stopReason != "" && stopReason != "tool_use" {
		return fail("messages_tools_stream", fmt.Sprintf("stop_reason is %q, want tool_use", stopReason))
	}
	return nil
}