package runner

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
	"github.com/beranekio/anthropic-compatibility-tester/internal/mockserver"
)

func TestRunAllPassesAgainstMockServer(t *testing.T) {
	server := mockserver.New()
	t.Cleanup(server.Close)

	cfg := &config.Config{
		BaseURL:         server.BaseURL(),
		APIKey:          "test-key",
		Model:           "claude-sonnet-4-6",
		CompletionModel: config.DefaultCompletionModel,
		VisionModel:     "claude-sonnet-4-6",
		RequestTimeout:  30 * time.Second,
		Suites: []string{
			"models",
			"models_get",
			"messages",
			"messages_stream",
			"messages_tools",
			"messages_tools_stream",
			"messages_json",
			"messages_multi_turn",
			"messages_count_tokens",
			"messages_vision",
			"completions",
			"completions_stream",
			"message_batches_create",
			"message_batches_get",
			"message_batches_cancel",
			"message_batches_list",
			"beta_files",
			"beta_skills",
			"beta_skill_versions",
			"error_responses",
		},
	}

	runner := New(cfg)
	runner.Output = &bytes.Buffer{}

	results, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code := ExitCode(results); code != 0 {
		t.Fatalf("ExitCode() = %d, want 0; summary:\n%s", code, FormatSummary(results))
	}
}

func TestRunAllFailsAgainstBrokenServer(t *testing.T) {
	server := mockserver.BrokenServer()
	t.Cleanup(server.Close)

	cfg := &config.Config{
		BaseURL:        server.BaseURL(),
		APIKey:         "test-key",
		Model:          "claude-sonnet-4-6",
		RequestTimeout: 10 * time.Second,
		Suites:         []string{"messages"},
	}

	runner := New(cfg)
	runner.Output = &bytes.Buffer{}

	results, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if code := ExitCode(results); code == 0 {
		t.Fatalf("ExitCode() = 0, want non-zero against broken server")
	}
}

func TestHandlerServesMessages(t *testing.T) {
	ts := httptest.NewServer(mockserver.Handler())
	t.Cleanup(ts.Close)

	body := `{"model":"claude-sonnet-4-6","max_tokens":16,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`
	resp, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want 200", resp.StatusCode)
	}
}