package mockserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrokenServerReturnsJSONErrorWithContentType(t *testing.T) {
	server := BrokenServer()
	t.Cleanup(server.Close)

	resp, err := http.Post(server.URL+"/v1/messages", "application/json", strings.NewReader(`{"model":"claude-sonnet-4-6"}`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d, want 400", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
}

func TestHandlerServesMessages(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	body := `{"model":"claude-sonnet-4-6","max_tokens":16,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`
	resp, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want 200, body = %s", resp.StatusCode, payload)
	}
	if !strings.Contains(string(payload), `"type":"message"`) && !strings.Contains(string(payload), `"type": "message"`) {
		t.Fatalf("response body = %s, want a message object", payload)
	}
}