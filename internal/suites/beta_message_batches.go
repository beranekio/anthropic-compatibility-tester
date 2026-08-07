package suites

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// BetaMessageBatchesCreate verifies POST /v1/messages/batches?beta=true.
type BetaMessageBatchesCreate struct{}

func (BetaMessageBatchesCreate) Name() string { return "beta_message_batches_create" }
func (BetaMessageBatchesCreate) Description() string {
	return "Beta message batch create (POST /v1/messages/batches?beta=true)"
}

func (BetaMessageBatchesCreate) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var batchID string
	defer func() { cleanupBetaMessageBatch(client, batchID) }()

	created, err := client.Beta.Messages.Batches.New(ctx, anthropic.BetaMessageBatchNewParams{
		Requests: []anthropic.BetaMessageBatchNewParamsRequest{{
			CustomID: "batch-request-1",
			Params: anthropic.BetaMessageBatchNewParamsRequestParams{
				Model:     anthropic.Model(cfg.Model),
				MaxTokens: 64,
				Messages: []anthropic.BetaMessageParam{
					anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Reply with exactly the word: pong")),
				},
			},
		}},
	})
	if err != nil {
		return fmt.Errorf("beta message batch create failed: %w", err)
	}
	if err := validateBetaMessageBatchObject("beta_message_batches_create", created); err != nil {
		return err
	}
	batchID = created.ID
	if created.ProcessingStatus != anthropic.BetaMessageBatchProcessingStatusInProgress &&
		created.ProcessingStatus != anthropic.BetaMessageBatchProcessingStatusEnded {
		return fail("beta_message_batches_create", fmt.Sprintf("processing_status is %q, want in_progress or ended", created.ProcessingStatus))
	}
	return nil
}

func validateBetaMessageBatchObject(suite string, batch *anthropic.BetaMessageBatch) error {
	if batch == nil {
		return fail(suite, "response is nil")
	}
	if batch.ID == "" {
		return fail(suite, "batch missing id")
	}
	if string(batch.Type) != "message_batch" {
		return fail(suite, fmt.Sprintf("batch type is %q, want message_batch", batch.Type))
	}
	if batch.ProcessingStatus == "" {
		return fail(suite, "batch missing processing_status")
	}
	return nil
}

func cleanupBetaMessageBatch(client anthropic.Client, batchID string) {
	if batchID == "" {
		return
	}
	cancelCtx, cancelCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, _ = client.Beta.Messages.Batches.Cancel(cancelCtx, batchID, anthropic.BetaMessageBatchCancelParams{})
	cancelCancel()

	deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, _ = client.Beta.Messages.Batches.Delete(deleteCtx, batchID, anthropic.BetaMessageBatchDeleteParams{})
	deleteCancel()
}
