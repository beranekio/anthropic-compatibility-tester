package suites

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	ticker := time.NewTicker(messageBatchPollInterval)
	defer ticker.Stop()
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
		case <-ticker.C:
		}
	}
}

// waitForMessageBatchCancelable polls until the batch is in_progress (cancelable),
// already canceling, or ended. Returns skipCancel=true when cancel is unnecessary.
func waitForMessageBatchCancelable(ctx context.Context, client anthropic.Client, suite, batchID string) (skipCancel bool, err error) {
	ticker := time.NewTicker(messageBatchPollInterval)
	defer ticker.Stop()
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
		case <-ticker.C:
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

func isMessageBatchCancelAlreadyTerminalError(apiErr *anthropic.Error) bool {
	switch apiErr.StatusCode {
	case http.StatusConflict, http.StatusBadRequest:
		return true
	default:
		return false
	}
}

func isMessageBatchCancelTerminalStatus(status anthropic.MessageBatchProcessingStatus) bool {
	return status == anthropic.MessageBatchProcessingStatusCanceling ||
		status == anthropic.MessageBatchProcessingStatusEnded
}

func confirmMessageBatchCancelTerminalState(ctx context.Context, client anthropic.Client, suite, batchID string) error {
	got, err := client.Messages.Batches.Get(ctx, batchID)
	if err != nil {
		return fmt.Errorf("message batch get failed: %w", err)
	}
	if err := validateMessageBatchObject(suite, got); err != nil {
		return err
	}
	if got.ID != batchID {
		return fail(suite, fmt.Sprintf("batch id is %q, want %q", got.ID, batchID))
	}
	if !isMessageBatchCancelTerminalStatus(got.ProcessingStatus) {
		return fail(suite, fmt.Sprintf("cancel returned terminal error but processing_status is %q, want canceling or ended", got.ProcessingStatus))
	}
	return nil
}

// exerciseMessageBatchCancelEndpoint calls Cancel when the batch already ended before
// becoming cancelable. Terminal-state errors only pass after GET confirms canceling/ended.
func exerciseMessageBatchCancelEndpoint(ctx context.Context, client anthropic.Client, suite, batchID string) error {
	canceled, err := client.Messages.Batches.Cancel(ctx, batchID)
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) && isMessageBatchCancelAlreadyTerminalError(apiErr) {
			return confirmMessageBatchCancelTerminalState(ctx, client, suite, batchID)
		}
		return fmt.Errorf("message batch cancel failed: %w", err)
	}
	if err := validateMessageBatchObject(suite, canceled); err != nil {
		return err
	}
	if canceled.ID != batchID {
		return fail(suite, fmt.Sprintf("batch id is %q, want %q", canceled.ID, batchID))
	}
	if !isMessageBatchCancelTerminalStatus(canceled.ProcessingStatus) {
		return fail(suite, fmt.Sprintf("processing_status is %q, want canceling or ended", canceled.ProcessingStatus))
	}
	return nil
}

// MessageBatchesCreate verifies POST /v1/messages/batches.
type MessageBatchesCreate struct{}

func (MessageBatchesCreate) Name() string { return "message_batches_create" }
func (MessageBatchesCreate) Description() string {
	return "Message batch create (POST /v1/messages/batches)"
}

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

func (MessageBatchesGet) Name() string { return "message_batches_get" }
func (MessageBatchesGet) Description() string {
	return "Message batch get (GET /v1/messages/batches/{id})"
}

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

func (MessageBatchesCancel) Name() string { return "message_batches_cancel" }
func (MessageBatchesCancel) Description() string {
	return "Message batch cancel (POST /v1/messages/batches/{id}/cancel)"
}

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

	skipCancel, err := waitForMessageBatchCancelable(ctx, client, "message_batches_cancel", batchID)
	if err != nil {
		return err
	}
	if skipCancel {
		return exerciseMessageBatchCancelEndpoint(ctx, client, "message_batches_cancel", batchID)
	}

	canceled, err := client.Messages.Batches.Cancel(ctx, batchID)
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) && isMessageBatchCancelAlreadyTerminalError(apiErr) {
			return exerciseMessageBatchCancelEndpoint(ctx, client, "message_batches_cancel", batchID)
		}
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

func (MessageBatchesList) Name() string { return "message_batches_list" }
func (MessageBatchesList) Description() string {
	return "Message batch list (GET /v1/messages/batches)"
}

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
	for i := range page.Data {
		item := &page.Data[i]
		if item.ID == batchID {
			if err := validateMessageBatchObject("message_batches_list", item); err != nil {
				return err
			}
			found = true
			break
		}
	}
	if !found {
		return fail("message_batches_list", "created batch missing from list response")
	}
	return nil
}

// MessageBatchesResults verifies GET /v1/messages/batches/{id}/results (JSONL stream).
type MessageBatchesResults struct{}

func (MessageBatchesResults) Name() string { return "message_batches_results" }
func (MessageBatchesResults) Description() string {
	return "Message batch results stream (GET /v1/messages/batches/{id}/results)"
}

func (MessageBatchesResults) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	const customID = "batch-request-1"
	var batchID string
	defer func() { cleanupMessageBatch(client, batchID) }()

	created, err := client.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{
		Requests: []anthropic.MessageBatchNewParamsRequest{{
			CustomID: customID,
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
	if err := validateMessageBatchObject("message_batches_results", created); err != nil {
		return err
	}
	batchID = created.ID

	// Results are only available after processing ends.
	if created.ProcessingStatus != anthropic.MessageBatchProcessingStatusEnded {
		skipCancel, err := waitForMessageBatchCancelable(ctx, client, "message_batches_results", batchID)
		if err != nil {
			return err
		}
		if !skipCancel {
			if _, err := client.Messages.Batches.Cancel(ctx, batchID); err != nil {
				var apiErr *anthropic.Error
				if !errors.As(err, &apiErr) || !isMessageBatchCancelAlreadyTerminalError(apiErr) {
					return fmt.Errorf("message batch cancel failed: %w", err)
				}
			}
		}
		if _, err := waitForMessageBatchStatus(ctx, client, "message_batches_results", batchID, func(status anthropic.MessageBatchProcessingStatus) bool {
			return status == anthropic.MessageBatchProcessingStatusEnded
		}); err != nil {
			return err
		}
	}

	stream := client.Messages.Batches.ResultsStreaming(ctx, batchID)
	defer stream.Close()

	count := 0
	foundCustomID := false
	for stream.Next() {
		item := stream.Current()
		count++
		if item.CustomID == "" {
			return fail("message_batches_results", "result line missing custom_id")
		}
		if item.CustomID == customID {
			foundCustomID = true
		}
		if item.Result.Type == "" {
			return fail("message_batches_results", "result line missing result.type")
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("message batch results stream failed: %w", err)
	}
	if count == 0 {
		return fail("message_batches_results", "results stream returned no lines")
	}
	if !foundCustomID {
		return fail("message_batches_results", fmt.Sprintf("no result line with custom_id %q", customID))
	}
	return nil
}

// MessageBatchesDelete verifies DELETE /v1/messages/batches/{id}.
type MessageBatchesDelete struct{}

func (MessageBatchesDelete) Name() string { return "message_batches_delete" }
func (MessageBatchesDelete) Description() string {
	return "Message batch delete (DELETE /v1/messages/batches/{id})"
}

func (MessageBatchesDelete) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	var batchID string
	// On success we delete in the suite; defer is a best-effort cleanup if we fail mid-way.
	defer func() {
		if batchID != "" {
			cleanupMessageBatch(client, batchID)
		}
	}()

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
	if err := validateMessageBatchObject("message_batches_delete", created); err != nil {
		return err
	}
	batchID = created.ID

	if created.ProcessingStatus != anthropic.MessageBatchProcessingStatusEnded {
		skipCancel, err := waitForMessageBatchCancelable(ctx, client, "message_batches_delete", batchID)
		if err != nil {
			return err
		}
		if !skipCancel {
			if _, err := client.Messages.Batches.Cancel(ctx, batchID); err != nil {
				var apiErr *anthropic.Error
				if !errors.As(err, &apiErr) || !isMessageBatchCancelAlreadyTerminalError(apiErr) {
					return fmt.Errorf("message batch cancel failed: %w", err)
				}
			}
		}
		if _, err := waitForMessageBatchStatus(ctx, client, "message_batches_delete", batchID, func(status anthropic.MessageBatchProcessingStatus) bool {
			return status == anthropic.MessageBatchProcessingStatusEnded
		}); err != nil {
			return err
		}
	}

	deleted, err := client.Messages.Batches.Delete(ctx, batchID)
	if err != nil {
		return fmt.Errorf("message batch delete failed: %w", err)
	}
	if deleted == nil {
		return fail("message_batches_delete", "delete response is nil")
	}
	if deleted.ID == "" {
		return fail("message_batches_delete", "delete response missing id")
	}
	if deleted.ID != batchID {
		return fail("message_batches_delete", fmt.Sprintf("delete id is %q, want %q", deleted.ID, batchID))
	}
	if string(deleted.Type) != "message_batch_deleted" {
		return fail("message_batches_delete", fmt.Sprintf("delete type is %q, want message_batch_deleted", deleted.Type))
	}
	batchID = "" // successful delete; skip cleanup
	return nil
}
