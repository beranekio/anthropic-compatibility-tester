package mockserver

import "time"

const defaultModelID = "claude-sonnet-4-6"

func capabilitySupport(supported bool) map[string]any {
	return map[string]any{"supported": supported}
}

func mockModelCapabilities() map[string]any {
	return map[string]any{
		"batch":              capabilitySupport(true),
		"citations":          capabilitySupport(true),
		"code_execution":     capabilitySupport(true),
		"image_input":        capabilitySupport(true),
		"pdf_input":          capabilitySupport(true),
		"structured_outputs": capabilitySupport(true),
		"context_management": map[string]any{
			"supported":                 true,
			"clear_thinking_20251015":   capabilitySupport(true),
			"clear_tool_uses_20250919":  capabilitySupport(true),
			"compact_20260112":          capabilitySupport(true),
		},
		"effort": map[string]any{
			"supported": true,
			"low":       capabilitySupport(true),
			"medium":    capabilitySupport(true),
			"high":      capabilitySupport(true),
			"max":       capabilitySupport(true),
			"xhigh":     capabilitySupport(true),
		},
		"thinking": map[string]any{
			"supported": true,
			"types": map[string]any{
				"adaptive": capabilitySupport(true),
				"enabled":  capabilitySupport(true),
			},
		},
	}
}

func mockModelPayload(id string) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"id":               id,
		"type":             "model",
		"display_name":     "Claude Sonnet 4.6",
		"created_at":       now,
		"max_input_tokens": 200000,
		"max_tokens":       8192,
		"capabilities":     mockModelCapabilities(),
	}
}

func mockMessagePayload(text string, stopReason string, content []map[string]any) map[string]any {
	if content == nil {
		content = []map[string]any{{"type": "text", "text": text}}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"id":            "msg_mock_1",
		"type":          "message",
		"role":          "assistant",
		"model":         defaultModelID,
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  10,
			"output_tokens": 5,
		},
		"container": map[string]any{
			"id":         "ctr_mock_1",
			"expires_at": now,
		},
		"stop_details": map[string]any{
			"type":        "refusal",
			"category":    "cyber",
			"explanation": "",
		},
	}
}

func mockMessageBatchPayload(id, status string) map[string]any {
	now := time.Now().UTC()
	created := now.Format(time.RFC3339)
	expires := now.Add(24 * time.Hour).Format(time.RFC3339)
	ended := ""
	cancelInitiated := ""
	if status == "ended" {
		ended = created
	}
	if status == "canceling" {
		cancelInitiated = created
	}
	return map[string]any{
		"id":                  id,
		"type":                "message_batch",
		"processing_status":   status,
		"created_at":          created,
		"expires_at":          expires,
		"archived_at":         "",
		"ended_at":            ended,
		"cancel_initiated_at": cancelInitiated,
		"results_url":         "https://mock.example/batches/" + id + "/results",
		"request_counts": map[string]any{
			"processing": 0,
			"succeeded":  1,
			"errored":    0,
			"canceled":   0,
			"expired":    0,
		},
	}
}

func mockCompletionPayload(text string) map[string]any {
	return map[string]any{
		"id":           "cmpl_mock_1",
		"type":         "completion",
		"model":        defaultModelID,
		"completion":   text,
		"stop_reason":  "stop_sequence",
	}
}

func mockFileMetadata(id, filename string, size int64) map[string]any {
	return map[string]any{
		"id":           id,
		"type":         "file",
		"filename":     filename,
		"mime_type":    "text/plain",
		"size_bytes":   size,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
		"downloadable": true,
	}
}

func mockSkillPayload(id string) map[string]any {
	return map[string]any{
		"id":             id,
		"type":           "skill",
		"created_at":     time.Now().UTC().Format(time.RFC3339),
		"display_title":  "Compatibility Test Skill",
		"latest_version": "1759178010641129",
		"source":         "custom",
	}
}

func mockSkillVersionPayload(skillID, version string) map[string]any {
	return map[string]any{
		"id":          "skver_mock_" + version,
		"type":        "skill_version",
		"version":     version,
		"skill_id":    skillID,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"name":        "compatibility-test-skill",
		"description": "compatibility test skill",
		"directory":   "compatibility-test-skill",
	}
}

func anthropicErrorPayload(message, errType string) map[string]any {
	return map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errType,
			"message": message,
		},
	}
}