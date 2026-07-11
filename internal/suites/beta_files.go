package suites

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
	"github.com/beranekio/anthropic-compatibility-tester/internal/testutil"
)

// BetaFiles verifies the Beta Files API lifecycle via client.Beta.Files.*.
type BetaFiles struct{}

func (BetaFiles) Name() string { return "beta_files" }
func (BetaFiles) Description() string {
	return "Beta Files API lifecycle (POST/GET/DELETE /v1/files?beta=true, GET /v1/files/{id}/content?beta=true)"
}

func (BetaFiles) Run(ctx context.Context, client anthropic.Client, _ *config.Config) error {
	deleted := false
	var fileID string
	defer func() {
		if fileID != "" && !deleted {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = client.Beta.Files.Delete(cleanupCtx, fileID, anthropic.BetaFileDeleteParams{})
		}
	}()

	uploaded, err := client.Beta.Files.Upload(ctx, anthropic.BetaFileUploadParams{
		File: testutil.SmallTextFileReader(),
	})
	if err != nil {
		return fmt.Errorf("beta file upload failed: %w", err)
	}
	if err := validateFileMetadata("beta_files", uploaded); err != nil {
		return err
	}
	fileID = uploaded.ID

	listPage, err := client.Beta.Files.List(ctx, anthropic.BetaFileListParams{})
	if err != nil {
		return fmt.Errorf("beta file list failed: %w", err)
	}
	if listPage == nil {
		return fail("beta_files", "list response is nil")
	}
	found := false
	for i := range listPage.Data {
		item := &listPage.Data[i]
		if item.ID == fileID {
			if err := validateFileMetadata("beta_files", item); err != nil {
				return err
			}
			found = true
			break
		}
	}
	if !found {
		return fail("beta_files", "uploaded file missing from list response")
	}

	got, err := client.Beta.Files.GetMetadata(ctx, fileID, anthropic.BetaFileGetMetadataParams{})
	if err != nil {
		return fmt.Errorf("beta file get metadata failed: %w", err)
	}
	if err := validateFileMetadata("beta_files", got); err != nil {
		return err
	}
	if got.ID != fileID {
		return fail("beta_files", fmt.Sprintf("get id is %q, want %q", got.ID, fileID))
	}

	resp, err := client.Beta.Files.Download(ctx, fileID, anthropic.BetaFileDownloadParams{})
	if err != nil {
		return fmt.Errorf("beta file download failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("beta file download read failed: %w", err)
	}
	if err := validateFileContentBody("beta_files", body, testutil.SmallTextFileBytes()); err != nil {
		return err
	}

	deletedResp, err := client.Beta.Files.Delete(ctx, fileID, anthropic.BetaFileDeleteParams{})
	if err != nil {
		return fmt.Errorf("beta file delete failed: %w", err)
	}
	if err := validateDeletedFile("beta_files", deletedResp, fileID); err != nil {
		return err
	}
	deleted = true
	return nil
}

func validateDeletedFile(suite string, deleted *anthropic.DeletedFile, wantID string) error {
	if deleted == nil {
		return fail(suite, "delete response is nil")
	}
	if deleted.ID == "" {
		return fail(suite, "delete response missing id")
	}
	if deleted.ID != wantID {
		return fail(suite, fmt.Sprintf("delete id is %q, want %q", deleted.ID, wantID))
	}
	if deleted.Type != anthropic.DeletedFileTypeFileDeleted {
		return fail(suite, fmt.Sprintf("delete type is %q, want file_deleted", deleted.Type))
	}
	return nil
}

func validateFileMetadata(suite string, file *anthropic.FileMetadata) error {
	if file == nil {
		return fail(suite, "file metadata is nil")
	}
	if file.ID == "" {
		return fail(suite, "file missing id")
	}
	if file.Filename == "" {
		return fail(suite, "file missing filename")
	}
	if file.MimeType == "" {
		return fail(suite, "file missing mime_type")
	}
	if string(file.Type) != "file" {
		return fail(suite, fmt.Sprintf("file type is %q, want file", file.Type))
	}
	return nil
}

func validateFileContentBody(suite string, body, want []byte) error {
	if len(body) != len(want) {
		return fail(suite, fmt.Sprintf("content has %d bytes, want %d", len(body), len(want)))
	}
	if !bytes.Equal(body, want) {
		return fail(suite, "content body does not match uploaded file")
	}
	return nil
}