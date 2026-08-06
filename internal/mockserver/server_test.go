package mockserver

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
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

func TestHandlerRejectsInvalidJSONMessages(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(`{not-json`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d, want 400", resp.StatusCode)
	}
}

func TestHandlerRejectsInvalidJSONCompletions(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/v1/complete", "application/json", strings.NewReader(`{not-json`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d, want 400", resp.StatusCode)
	}
}

func TestHandlerRejectsSkillCreateWithoutFiles(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Post(ts.URL+"/v1/skills", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d, want 400", resp.StatusCode)
	}
}

func TestHandlerRejectsSkillCreateWithTextFileFieldOnly(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("file", "not-a-real-file-part"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("multipart Close() error = %v", err)
	}

	resp, err := http.Post(ts.URL+"/v1/skills", w.FormDataContentType(), &body)
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d, want 400 (text form field is not a skill file part)", resp.StatusCode)
	}
}

func TestHandlerRejectsSkillVersionCreateWithoutFiles(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	// Create a skill first so a missing-files version create is not masked by 404.
	var createBody bytes.Buffer
	createW := multipart.NewWriter(&createBody)
	part, err := createW.CreateFormFile("files", "compatibility-test-skill/SKILL.md")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := io.WriteString(part, "---\nname: test\ndescription: test\n---\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := createW.Close(); err != nil {
		t.Fatalf("multipart Close() error = %v", err)
	}
	createResp, err := http.Post(ts.URL+"/v1/skills", createW.FormDataContentType(), &createBody)
	if err != nil {
		t.Fatalf("skill create http.Post() error = %v", err)
	}
	createPayload, err := io.ReadAll(createResp.Body)
	_ = createResp.Body.Close()
	if err != nil {
		t.Fatalf("read create body error = %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("skill create status = %d, want 200, body = %s", createResp.StatusCode, createPayload)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createPayload, &created); err != nil || created.ID == "" {
		t.Fatalf("skill create response missing id: %s (err=%v)", createPayload, err)
	}

	resp, err := http.Post(ts.URL+"/v1/skills/"+created.ID+"/versions", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("version create http.Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("version create status = %d, want 400 (missing skill files)", resp.StatusCode)
	}
}

func TestMessageStreamStartsWithEmptyEnvelope(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	body := `{"model":"claude-sonnet-4-6","max_tokens":16,"stream":true,"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`
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
	text := string(payload)
	if !strings.Contains(text, `"type":"message_start"`) && !strings.Contains(text, `"type": "message_start"`) {
		t.Fatalf("stream missing message_start: %s", text)
	}
	// Final content must not be embedded in message_start; deltas deliver text.
	if strings.Contains(text, `"text":"one two three"`) {
		t.Fatalf("message_start should not include final text content: %s", text)
	}
	if !strings.Contains(text, `"type":"text_delta"`) && !strings.Contains(text, `"type": "text_delta"`) {
		t.Fatalf("stream missing text_delta events: %s", text)
	}
}
