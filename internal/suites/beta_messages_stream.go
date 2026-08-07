package suites

import (
	"context"
	"fmt"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// BetaMessagesStream verifies streaming POST /v1/messages?beta=true.
type BetaMessagesStream struct{}

func (BetaMessagesStream) Name() string { return "beta_messages_stream" }
func (BetaMessagesStream) Description() string {
	return "Beta streaming message completion (POST /v1/messages?beta=true, stream=true)"
}

func (BetaMessagesStream) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var httpResp *http.Response
	stream := client.Beta.Messages.NewStreaming(ctx, anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 64,
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Count from one to three.")),
		},
	}, option.WithResponseInto(&httpResp))
	defer stream.Close()

	if err := stream.Err(); err != nil {
		return fmt.Errorf("beta messages stream failed: %w", err)
	}
	if err := validateEventStreamContentType("beta_messages_stream", httpResp); err != nil {
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
		return fmt.Errorf("beta messages stream failed: %w", err)
	}
	if events == 0 {
		return fail("beta_messages_stream", "stream returned no events")
	}
	if !hasMessageStart {
		return fail("beta_messages_stream", "stream missing message_start event")
	}
	if !finished {
		return fail("beta_messages_stream", "stream missing terminal message_stop event")
	}
	if stopReason == "" {
		return fail("beta_messages_stream", "stream missing stop_reason in message_delta")
	}
	if stopReason == "refusal" {
		return nil
	}
	if !hasOutput {
		return fail("beta_messages_stream", "stream produced no text content")
	}
	return nil
}
