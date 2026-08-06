package suites

import (
	"context"
	"fmt"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// MessagesStream verifies streaming messages.
type MessagesStream struct{}

func (MessagesStream) Name() string { return "messages_stream" }
func (MessagesStream) Description() string {
	return "Streaming message completion (POST /v1/messages, stream=true)"
}

func (MessagesStream) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var httpResp *http.Response
	stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 64,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Count from one to three.")),
		},
	}, option.WithResponseInto(&httpResp))
	defer stream.Close()

	if err := stream.Err(); err != nil {
		return fmt.Errorf("messages stream failed: %w", err)
	}
	if err := validateEventStreamContentType("messages_stream", httpResp); err != nil {
		return err
	}

	events := 0
	var hasMessageStart bool
	// Track the currently open content block (Anthropic streams blocks sequentially),
	// similar to messages_tools_stream's inToolUse. Global "ever seen" flags would allow
	// text deltas from a later block to borrow start/stop events from an earlier one.
	var inContentBlock bool
	var textInOpenBlock bool
	var completedTextBlock bool
	var hasOutput bool
	var finished bool
	var stopReason string
	for stream.Next() {
		event := stream.Current()
		events++
		switch event.Type {
		case "message_start":
			start := event.AsMessageStart()
			if err := validateMessageStreamStartEnvelope("messages_stream", &start.Message); err != nil {
				return err
			}
			hasMessageStart = true
		case "content_block_start":
			if inContentBlock {
				return fail("messages_stream", "content_block_start while a content block is already open")
			}
			inContentBlock = true
			textInOpenBlock = false
		case "content_block_delta":
			if !inContentBlock {
				return fail("messages_stream", "content_block_delta without an open content_block_start")
			}
			delta := event.AsContentBlockDelta().Delta
			if delta.Text != "" {
				hasOutput = true
				textInOpenBlock = true
			}
		case "content_block_stop":
			if !inContentBlock {
				return fail("messages_stream", "content_block_stop without an open content_block_start")
			}
			if textInOpenBlock {
				completedTextBlock = true
			}
			inContentBlock = false
			textInOpenBlock = false
		case "message_stop":
			finished = true
		case "message_delta":
			if event.AsMessageDelta().Delta.StopReason != "" {
				stopReason = string(event.AsMessageDelta().Delta.StopReason)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("messages stream failed: %w", err)
	}
	if events == 0 {
		return fail("messages_stream", "stream returned no events")
	}
	if err := validateMessageStreamCompleted("messages_stream", finished, hasMessageStart, hasOutput, stopReason); err != nil {
		return err
	}
	// Text-only request: tool_use stop_reason is not a valid compatibility outcome.
	if stopReason == "tool_use" {
		return fail("messages_stream", "stop_reason is tool_use on text-only stream")
	}
	if inContentBlock {
		return fail("messages_stream", "stream ended with an open content block (missing content_block_stop)")
	}
	if hasOutput && !completedTextBlock {
		return fail("messages_stream", "stream text deltas missing content_block start/stop lifecycle on the same block")
	}
	return nil
}
