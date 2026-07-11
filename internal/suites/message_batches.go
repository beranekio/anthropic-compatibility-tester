package suites

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

const messageBatchPollInterval = 2 * time.Second

func validateMessageBatchObject(suite string, batch *anthropic.MessageBatch) error {
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

func waitForMessageBatchStatus(ctx context.Context, client anthropic.Client, suite, batchID string, accept func(anthropic.MessageBatchProcessingStatus) bool) (*anthropic.MessageBatch, error) {
	for {
		got, err := client.Messages.Batches.Get(ctx, batchID)
		if err != nil {
			return nil, fmt.Errorf("message batch get failed: %w", err)
		}
		if err := validateMessageBatchObject(suite, got); err != nil {
			return nil, err
		}
		if got.ID != batchID {
			return nil, fail(suite, fmt.Sprintf("batch id is %q, want %q", got.ID, batchID))
		}
		if accept(got.ProcessingStatus) {
			return got, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for message batch status: %w", ctx.Err())
		case <-time.After(messageBatchPollInterval):
		}
	}
}

// waitForMessageBatchCancelable polls until the batch is in_progress (cancelable),
// already canceling, or ended. Returns skipCancel=true when cancel is unnecessary.
func waitForMessageBatchCancelable(ctx context.Context, client anthropic.Client, suite, batchID string) (skipCancel bool, err error) {
	for {
		got, err := client.Messages.Batches.Get(ctx, batchID)
		if err != nil {
			return false, fmt.Errorf("message batch get failed: %w", err)
		}
		if err := validateMessageBatchObject(suite, got); err != nil {
			return false, err
		}
		if got.ID != batchID {
			return false, fail(suite, fmt.Sprintf("batch id is %q, want %q", got.ID, batchID))
		}
		switch got.ProcessingStatus {
		case anthropic.MessageBatchProcessingStatusInProgress:
			return false, nil
		case anthropic.MessageBatchProcessingStatusCanceling,
			anthropic.MessageBatchProcessingStatusEnded:
			return true, nil
		}
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("timed out waiting for cancelable message batch status: %w", ctx.Err())
		case <-time.After(messageBatchPollInterval):
		}
	}
}

func cleanupMessageBatch(client anthropic.Client, batchID string) {
	if batchID == "" {
		return
	}

	cancelableCtx, cancelableCancel := context.WithTimeout(context.Background(), 10*time.Second)
	skipCancel, err := waitForMessageBatchCancelable(cancelableCtx, client, "message_batches", batchID)
	cancelableCancel()
	if err == nil && !skipCancel {
		cancelCtx, cancelCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, _ = client.Messages.Batches.Cancel(cancelCtx, batchID)
		cancelCancel()
	}

	endedCtx, endedCancel := context.WithTimeout(context.Background(), 30*time.Second)
	_, _ = waitForMessageBatchStatus(endedCtx, client, "message_batches", batchID, func(status anthropic.MessageBatchProcessingStatus) bool {
		return status == anthropic.MessageBatchProcessingStatusEnded
	})
	endedCancel()

	deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, _ = client.Messages.Batches.Delete(deleteCtx, batchID)
	deleteCancel()
}

// MessageBatchesCreate verifies POST /v1/messages/batches.
type MessageBatchesCreate struct{}

func (MessageBatchesCreate) Name() string        { return "message_batches_create" }
func (MessageBatchesCreate) Description() string { return "Message batch create (POST /v1/messages/batches)" }

func (MessageBatchesCreate) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var batchID string
	defer func() { cleanupMessageBatch(client, batchID) }()

	created, err := client.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{
		Requests: []anthropic.MessageBatchNewParamsRequest{{
			CustomID: "batch-request-1",
			Params: anthropic.MessageBatchNewParamsRequestParams{
				Model:     anthropic.Model(cfg.Model),
				MaxTokens: 64,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with exactly the word: pong")),
				},
			},
		}},
	})
	if err != nil {
		return fmt.Errorf("message batch create failed: %w", err)
	}
	if err := validateMessageBatchObject("message_batches_create", created); err != nil {
		return err
	}
	batchID = created.ID
	if created.ProcessingStatus != anthropic.MessageBatchProcessingStatusInProgress &&
		created.ProcessingStatus != anthropic.MessageBatchProcessingStatusEnded {
		return fail("message_batches_create", fmt.Sprintf("processing_status is %q, want in_progress or ended", created.ProcessingStatus))
	}
	return nil
}

// MessageBatchesGet verifies GET /v1/messages/batches/{id}.
type MessageBatchesGet struct{}

func (MessageBatchesGet) Name() string        { return "message_batches_get" }
func (MessageBatchesGet) Description() string { return "Message batch get (GET /v1/messages/batches/{id})" }

func (MessageBatchesGet) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var batchID string
	defer func() { cleanupMessageBatch(client, batchID) }()

	created, err := client.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{
		Requests: []anthropic.MessageBatchNewParamsRequest{{
			CustomID: "batch-request-1",
			Params: anthropic.MessageBatchNewParamsRequestParams{
				Model:     anthropic.Model(cfg.Model),
				MaxTokens: 64,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with exactly the word: pong")),
				},
			},
		}},
	})
	if err != nil {
		return fmt.Errorf("message batch create failed: %w", err)
	}
	if err := validateMessageBatchObject("message_batches_get", created); err != nil {
		return err
	}
	batchID = created.ID

	got, err := client.Messages.Batches.Get(ctx, batchID)
	if err != nil {
		return fmt.Errorf("message batch get failed: %w", err)
	}
	if err := validateMessageBatchObject("message_batches_get", got); err != nil {
		return err
	}
	if got.ID != batchID {
		return fail("message_batches_get", fmt.Sprintf("batch id is %q, want %q", got.ID, batchID))
	}
	return nil
}

// MessageBatchesCancel verifies POST /v1/messages/batches/{id}/cancel.
type MessageBatchesCancel struct{}

func (MessageBatchesCancel) Name() string        { return "message_batches_cancel" }
func (MessageBatchesCancel) Description() string { return "Message batch cancel (POST /v1/messages/batches/{id}/cancel)" }

func (MessageBatchesCancel) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var batchID string
	defer func() { cleanupMessageBatch(client, batchID) }()

	created, err := client.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{
		Requests: []anthropic.MessageBatchNewParamsRequest{{
			CustomID: "batch-request-1",
			Params: anthropic.MessageBatchNewParamsRequestParams{
				Model:     anthropic.Model(cfg.Model),
				MaxTokens: 64,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with exactly the word: pong")),
				},
			},
		}},
	})
	if err != nil {
		return fmt.Errorf("message batch create failed: %w", err)
	}
	if err := validateMessageBatchObject("message_batches_cancel", created); err != nil {
		return err
	}
	batchID = created.ID

	canceled, err := client.Messages.Batches.Cancel(ctx, batchID)
	if err != nil {
		return fmt.Errorf("message batch cancel failed: %w", err)
	}
	if err := validateMessageBatchObject("message_batches_cancel", canceled); err != nil {
		return err
	}
	if canceled.ID != batchID {
		return fail("message_batches_cancel", fmt.Sprintf("batch id is %q, want %q", canceled.ID, batchID))
	}
	if canceled.ProcessingStatus != anthropic.MessageBatchProcessingStatusCanceling &&
		canceled.ProcessingStatus != anthropic.MessageBatchProcessingStatusEnded {
		return fail("message_batches_cancel", fmt.Sprintf("processing_status is %q, want canceling or ended", canceled.ProcessingStatus))
	}
	return nil
}

// MessageBatchesList verifies GET /v1/messages/batches.
type MessageBatchesList struct{}

func (MessageBatchesList) Name() string        { return "message_batches_list" }
func (MessageBatchesList) Description() string { return "Message batch list (GET /v1/messages/batches)" }

func (MessageBatchesList) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var batchID string
	defer func() { cleanupMessageBatch(client, batchID) }()

	created, err := client.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{
		Requests: []anthropic.MessageBatchNewParamsRequest{{
			CustomID: "batch-request-1",
			Params: anthropic.MessageBatchNewParamsRequestParams{
				Model:     anthropic.Model(cfg.Model),
				MaxTokens: 64,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(anthropic.NewTextBlock("Reply with exactly the word: pong")),
				},
			},
		}},
	})
	if err != nil {
		return fmt.Errorf("message batch create failed: %w", err)
	}
	if err := validateMessageBatchObject("message_batches_list", created); err != nil {
		return err
	}
	batchID = created.ID

	page, err := client.Messages.Batches.List(ctx, anthropic.MessageBatchListParams{})
	if err != nil {
		return fmt.Errorf("message batch list failed: %w", err)
	}
	if page == nil {
		return fail("message_batches_list", "response is nil")
	}
	found := false
	for _, item := range page.Data {
		if item.ID == batchID {
			found = true
			break
		}
	}
	if !found {
		return fail("message_batches_list", "created batch missing from list response")
	}
	return nil
}