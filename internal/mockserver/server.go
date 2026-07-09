package mockserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// Server provides a minimal Anthropic-compatible HTTP API for CI tests.
type Server struct {
	*httptest.Server
	mux        *http.ServeMux
	batchStore *batchStore
	fileStore  *fileStore
	skillStore *skillStore
}

// New starts a mock Anthropic API server.
func New() *Server {
	s := newServerWithRoutes()
	s.Server = httptest.NewServer(s.mux)
	return s
}

// Handler returns an http.Handler serving the mock Anthropic API.
func Handler() http.Handler {
	return newServerWithRoutes().mux
}

// BaseURL returns the API base URL for SDK clients.
func (s *Server) BaseURL() string {
	return s.URL
}

func newServerWithRoutes() *Server {
	mux := http.NewServeMux()
	s := &Server{
		batchStore: newBatchStore(),
		fileStore:  newFileStore(),
		skillStore: newSkillStore(),
		mux:        mux,
	}

	mux.HandleFunc("GET /v1/models", handleModels)
	mux.HandleFunc("GET /v1/models/{id}", handleModelGet)
	mux.HandleFunc("POST /v1/messages", s.handleMessages)
	mux.HandleFunc("POST /v1/messages/count_tokens", handleCountTokens)
	mux.HandleFunc("POST /v1/complete", handleCompletions)
	mux.HandleFunc("POST /v1/messages/batches", s.handleMessageBatchCreate)
	mux.HandleFunc("GET /v1/messages/batches", s.handleMessageBatchList)
	mux.HandleFunc("GET /v1/messages/batches/{id}", s.handleMessageBatchGet)
	mux.HandleFunc("POST /v1/messages/batches/{id}/cancel", s.handleMessageBatchCancel)
	mux.HandleFunc("DELETE /v1/messages/batches/{id}", s.handleMessageBatchDelete)
	mux.HandleFunc("POST /v1/files", s.handleBetaFileUpload)
	mux.HandleFunc("GET /v1/files", s.handleBetaFileList)
	mux.HandleFunc("GET /v1/files/{id}", s.handleBetaFileGet)
	mux.HandleFunc("DELETE /v1/files/{id}", s.handleBetaFileDelete)
	mux.HandleFunc("GET /v1/files/{id}/content", s.handleBetaFileContent)
	mux.HandleFunc("POST /v1/skills", s.handleBetaSkillCreate)
	mux.HandleFunc("GET /v1/skills", s.handleBetaSkillList)
	mux.HandleFunc("GET /v1/skills/{id}", s.handleBetaSkillGet)
	mux.HandleFunc("DELETE /v1/skills/{id}", s.handleBetaSkillDelete)
	mux.HandleFunc("POST /v1/skills/{id}/versions", s.handleBetaSkillVersionCreate)
	mux.HandleFunc("GET /v1/skills/{id}/versions", s.handleBetaSkillVersionList)
	mux.HandleFunc("GET /v1/skills/{id}/versions/{version}", s.handleBetaSkillVersionGet)

	return s
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message, errType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, anthropicErrorPayload(message, errType))
}

func handleModels(w http.ResponseWriter, _ *http.Request) {
	model := mockModelPayload(defaultModelID)
	writeJSON(w, map[string]any{
		"data":     []map[string]any{model},
		"has_more": false,
		"first_id": defaultModelID,
		"last_id":  defaultModelID,
	})
}

func handleModelGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeJSON(w, mockModelPayload(id))
}

func handleCountTokens(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"input_tokens": 12})
}

type messageRequest struct {
	Model        string          `json:"model"`
	Stream       bool            `json:"stream"`
	Messages     []messageInput  `json:"messages"`
	Tools        []json.RawMessage `json:"tools"`
	OutputConfig *struct {
		Format *struct {
			Type string `json:"type"`
		} `json:"format"`
	} `json:"output_config"`
}

type messageInput struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req messageRequest
	_ = json.Unmarshal(body, &req)

	if req.Model == "oct-invalid-model" {
		writeError(w, http.StatusBadRequest, "model: "+req.Model+" not found", "invalid_request_error")
		return
	}

	if req.Stream {
		writeMessageStream(w, &req)
		return
	}

	writeMessageResponse(w, &req)
}

func writeMessageResponse(w http.ResponseWriter, req *messageRequest) {
	if len(req.Tools) > 0 && !messageRequestHasToolResult(req.Messages) {
		writeJSON(w, mockMessagePayload("", "tool_use", []map[string]any{{
			"type":  "tool_use",
			"id":    "toolu_mock_1",
			"name":  "get_weather",
			"input": map[string]any{"location": "San Francisco, CA"},
		}}))
		return
	}

	text := "pong"
	if messageRequestHasToolResult(req.Messages) {
		text = "72"
	} else if req.OutputConfig != nil && req.OutputConfig.Format != nil && req.OutputConfig.Format.Type == "json_schema" {
		text = `{"answer":"pong"}`
	} else if messageRequestHasImage(req.Messages) {
		text = "I see an image"
	}

	writeJSON(w, mockMessagePayload(text, "end_turn", nil))
}

func writeMessageStream(w http.ResponseWriter, req *messageRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)

	message := mockMessagePayload("one two three", "end_turn", nil)
	if len(req.Tools) > 0 && !messageRequestHasToolResult(req.Messages) {
		message = mockMessagePayload("", "tool_use", []map[string]any{{
			"type":  "tool_use",
			"id":    "toolu_mock_1",
			"name":  "get_weather",
			"input": map[string]any{"location": "San Francisco, CA"},
		}})
	}

	events := []map[string]any{
		{"type": "message_start", "message": message},
		{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}},
		{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": "one"}},
		{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": " two three"}},
		{"type": "content_block_stop", "index": 0},
		{"type": "message_delta", "delta": map[string]any{"stop_reason": message["stop_reason"], "stop_sequence": nil}},
		{"type": "message_stop"},
	}

	if len(req.Tools) > 0 && !messageRequestHasToolResult(req.Messages) {
		events = []map[string]any{
			{"type": "message_start", "message": message},
			{"type": "content_block_start", "index": 0, "content_block": map[string]any{
				"type": "tool_use", "id": "toolu_mock_1", "name": "get_weather", "input": map[string]any{},
			}},
			{"type": "content_block_delta", "index": 0, "delta": map[string]any{
				"type": "input_json_delta", "partial_json": `{"location":"San Francisco, CA"}`,
			}},
			{"type": "content_block_stop", "index": 0},
			{"type": "message_delta", "delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}},
			{"type": "message_stop"},
		}
	}

	for _, event := range events {
		data, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event["type"], string(data))
	}
}

func messageRequestHasToolResult(messages []messageInput) bool {
	for _, msg := range messages {
		var blocks []map[string]json.RawMessage
		if err := json.Unmarshal(msg.Content, &blocks); err == nil {
			for _, block := range blocks {
				var blockType string
				_ = json.Unmarshal(block["type"], &blockType)
				if blockType == "tool_result" {
					return true
				}
			}
		}
	}
	return false
}

func messageRequestHasImage(messages []messageInput) bool {
	for _, msg := range messages {
		var blocks []map[string]json.RawMessage
		if err := json.Unmarshal(msg.Content, &blocks); err == nil {
			for _, block := range blocks {
				var blockType string
				_ = json.Unmarshal(block["type"], &blockType)
				if blockType == "image" {
					return true
				}
			}
		}
	}
	return false
}

func handleCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &req)

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		chunks := []string{"one", " two", " three"}
		for _, chunk := range chunks {
			payload, _ := json.Marshal(mockCompletionPayload(chunk))
			_, _ = fmt.Fprintf(w, "event: completion\ndata: %s\n\n", string(payload))
		}
		final, _ := json.Marshal(map[string]any{
			"type": "completion", "completion": "", "stop_reason": "stop_sequence",
		})
		_, _ = fmt.Fprintf(w, "event: completion\ndata: %s\n\n", string(final))
		return
	}

	writeJSON(w, mockCompletionPayload("pong"))
}

func (s *Server) handleMessageBatchCreate(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.batchStore.create("in_progress"))
}

func (s *Server) handleMessageBatchGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	payload, ok := s.batchStore.get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "batch not found", "not_found_error")
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleMessageBatchList(w http.ResponseWriter, _ *http.Request) {
	items := s.batchStore.listAll()
	firstID, lastID := "", ""
	if len(items) > 0 {
		if id, ok := items[0]["id"].(string); ok {
			firstID = id
		}
		if id, ok := items[len(items)-1]["id"].(string); ok {
			lastID = id
		}
	}
	writeJSON(w, map[string]any{
		"data":     items,
		"has_more": false,
		"first_id": firstID,
		"last_id":  lastID,
	})
}

func (s *Server) handleMessageBatchCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	payload, ok := s.batchStore.update(id, "canceling")
	if !ok {
		writeError(w, http.StatusNotFound, "batch not found", "not_found_error")
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleMessageBatchDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.batchStore.delete(id) {
		writeError(w, http.StatusNotFound, "batch not found", "not_found_error")
		return
	}
	writeJSON(w, map[string]any{"id": id, "type": "message_batch_deleted"})
}

func (s *Server) handleBetaFileUpload(w http.ResponseWriter, r *http.Request) {
	filename, content := parseMultipartFile(r)
	if filename == "" {
		filename = "test.txt"
	}
	if len(content) == 0 {
		content = []byte("compatibility test file\n")
	}
	writeJSON(w, s.fileStore.create(filename, content))
}

func (s *Server) handleBetaFileList(w http.ResponseWriter, _ *http.Request) {
	items := s.fileStore.listAll()
	firstID, lastID := "", ""
	if len(items) > 0 {
		if id, ok := items[0]["id"].(string); ok {
			firstID = id
		}
		if id, ok := items[len(items)-1]["id"].(string); ok {
			lastID = id
		}
	}
	writeJSON(w, map[string]any{
		"data":     items,
		"has_more": false,
		"first_id": firstID,
		"last_id":  lastID,
	})
}

func (s *Server) handleBetaFileGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	payload, ok := s.fileStore.get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "file not found", "not_found_error")
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleBetaFileDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.fileStore.delete(id) {
		writeError(w, http.StatusNotFound, "file not found", "not_found_error")
		return
	}
	writeJSON(w, map[string]any{"id": id, "type": "file_deleted"})
}

func (s *Server) handleBetaFileContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	content, ok := s.fileStore.content(id)
	if !ok {
		writeError(w, http.StatusNotFound, "file not found", "not_found_error")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(content)
}

func (s *Server) handleBetaSkillCreate(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.skillStore.create())
}

func (s *Server) handleBetaSkillGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	payload, ok := s.skillStore.get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "skill not found", "not_found_error")
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleBetaSkillList(w http.ResponseWriter, _ *http.Request) {
	items := s.skillStore.listAll()
	writeJSON(w, map[string]any{
		"data":      items,
		"has_more":  false,
		"next_page": nil,
	})
}

func (s *Server) handleBetaSkillDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.skillStore.delete(id) {
		writeError(w, http.StatusNotFound, "skill not found", "not_found_error")
		return
	}
	writeJSON(w, map[string]any{"id": id, "type": "skill_deleted"})
}

func (s *Server) handleBetaSkillVersionCreate(w http.ResponseWriter, r *http.Request) {
	skillID := r.PathValue("id")
	payload, ok := s.skillStore.addVersion(skillID)
	if !ok {
		writeError(w, http.StatusNotFound, "skill not found", "not_found_error")
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleBetaSkillVersionList(w http.ResponseWriter, r *http.Request) {
	skillID := r.PathValue("id")
	items := s.skillStore.listVersions(skillID)
	writeJSON(w, map[string]any{
		"data":      items,
		"has_more":  false,
		"next_page": nil,
	})
}

func (s *Server) handleBetaSkillVersionGet(w http.ResponseWriter, r *http.Request) {
	skillID := r.PathValue("id")
	version := r.PathValue("version")
	payload, ok := s.skillStore.getVersion(skillID, version)
	if !ok {
		writeError(w, http.StatusNotFound, "skill version not found", "not_found_error")
		return
	}
	writeJSON(w, payload)
}

func parseMultipartFile(r *http.Request) (string, []byte) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return "", nil
	}
	for _, files := range r.MultipartForm.File {
		for _, header := range files {
			file, err := header.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				continue
			}
			return header.Filename, content
		}
	}
	if r.MultipartForm != nil {
		for key, values := range r.MultipartForm.Value {
			if strings.EqualFold(key, "file") && len(values) > 0 {
				return "test.txt", []byte(values[0])
			}
		}
	}
	return "", nil
}

// BrokenServer returns a server that responds with incompatible payloads.
func BrokenServer() *Server {
	mux := http.NewServeMux()
	s := &Server{}

	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"data": []any{}})
	})
	mux.HandleFunc("GET /v1/models/{id}", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"type": "model"})
	})
	mux.HandleFunc("POST /v1/messages", brokenIncompatibleHandler)
	mux.HandleFunc("POST /v1/messages/count_tokens", brokenIncompatibleHandler)

	s.Server = httptest.NewServer(mux)
	return s
}

func brokenIncompatibleHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(`{"unexpected":"shape"}`))
}