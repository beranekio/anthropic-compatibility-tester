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
		case "content_block_delta":
			delta := event.AsContentBlockDelta().Delta
			if delta.Text != "" {
				hasOutput = true
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
		return fmt.Errorf("messages stream failed: %w", err)
	}
	if events == 0 {
		return fail("messages_stream", "stream returned no events")
	}
	return validateMessageStreamCompleted("messages_stream", finished, hasMessageStart, hasOutput, stopReason)
}