package suites

import (
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

func validateEventStreamContentType(suite string, resp *http.Response) error {
	if resp == nil {
		return fail(suite, "stream response is nil")
	}
	contentType := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fail(suite, fmt.Sprintf("Content-Type %q is invalid: %v", contentType, err))
	}
	if mediaType != "text/event-stream" {
		return fail(suite, fmt.Sprintf("Content-Type is %q, want text/event-stream", strings.TrimSpace(contentType)))
	}
	return nil
}

func validateMessageStreamStartEnvelope(suite string, msg *anthropic.Message) error {
	if msg == nil {
		return fail(suite, "message_start message is nil")
	}
	if msg.ID == "" {
		return fail(suite, "message_start missing id")
	}
	if msg.Model == "" {
		return fail(suite, "message_start missing model")
	}
	if string(msg.Type) != "message" {
		return fail(suite, fmt.Sprintf("message_start type is %q, want message", msg.Type))
	}
	if string(msg.Role) != "assistant" {
		return fail(suite, fmt.Sprintf("message_start role is %q, want assistant", msg.Role))
	}
	return nil
}

func validateMessageStreamCompleted(suite string, finished bool, hasMessageStart bool, hasOutput bool, stopReason string) error {
	if !hasMessageStart {
		return fail(suite, "stream missing message_start event")
	}
	if !finished {
		return fail(suite, "stream missing terminal message_stop event")
	}
	if stopReason == "" {
		return fail(suite, "stream missing stop_reason in message_delta")
	}
	if !hasOutput && stopReason != "refusal" {
		return fail(suite, "stream produced no text content or tool_use")
	}
	return nil
}

func validateCompletionStreamCompleted(suite string, finished bool, hasOutput bool) error {
	if !finished {
		return fail(suite, "stream missing terminal completion event")
	}
	if !hasOutput {
		return fail(suite, "stream produced no completion text")
	}
	return nil
}

func validateCompletionEnvelope(suite string, completion *anthropic.Completion) error {
	if completion == nil {
		return fail(suite, "response is nil")
	}
	if completion.ID == "" {
		return fail(suite, "response missing id")
	}
	if completion.Model == "" {
		return fail(suite, "response missing model")
	}
	if completion.StopReason == "" {
		return fail(suite, "response missing stop_reason")
	}
	if string(completion.Type) != "completion" {
		return fail(suite, fmt.Sprintf("response type is %q, want completion", completion.Type))
	}
	if completion.Completion == "" {
		return fail(suite, "response missing completion text")
	}
	return nil
}