package suites

import (
	"context"
	"fmt"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// CompletionsStream verifies legacy streaming POST /v1/complete.
type CompletionsStream struct{}

func (CompletionsStream) Name() string { return "completions_stream" }
func (CompletionsStream) Description() string {
	return "(deprecated) Streaming text completion (POST /v1/complete, stream=true)"
}

func (CompletionsStream) Deprecated() bool { return true }

func (CompletionsStream) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var httpResp *http.Response
	stream := client.Completions.NewStreaming(ctx, anthropic.CompletionNewParams{
		Model:             anthropic.Model(cfg.CompletionModel),
		MaxTokensToSample: 64,
		Prompt:            "\n\nHuman: Count from one to three.\n\nAssistant:",
	}, option.WithResponseInto(&httpResp))
	defer stream.Close()

	if err := stream.Err(); err != nil {
		return fmt.Errorf("completions stream failed: %w", err)
	}
	if err := validateEventStreamContentType("completions_stream", httpResp); err != nil {
		return err
	}

	var hasOutput bool
	var finished bool
	for stream.Next() {
		chunk := stream.Current()
		if chunk.Completion != "" {
			hasOutput = true
		}
		if chunk.StopReason != "" {
			finished = true
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("completions stream failed: %w", err)
	}
	return validateCompletionStreamCompleted("completions_stream", finished, hasOutput)
}