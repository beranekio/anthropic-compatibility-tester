package suites

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/shared"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

const errorTriggerModel = "oct-invalid-model"

// ErrorResponses verifies that the endpoint returns parseable Anthropic error payloads.
type ErrorResponses struct{}

func (ErrorResponses) Name() string { return "error_responses" }
func (ErrorResponses) Description() string {
	return "Anthropic-compatible error responses (invalid request)"
}

func (ErrorResponses) Run(ctx context.Context, client anthropic.Client, _ *config.Config) error {
	_, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(errorTriggerModel),
		MaxTokens: 16,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("This request should fail.")),
		},
	})
	if err == nil {
		return fail("error_responses", "expected request to fail with an API error")
	}

	var apiErr *anthropic.Error
	if !errors.As(err, &apiErr) {
		return fail("error_responses", fmt.Sprintf("error is %T, want *anthropic.Error", err))
	}
	return validateErrorResponseAPIError("error_responses", apiErr)
}

func validateErrorResponseAPIError(suite string, apiErr *anthropic.Error) error {
	if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
		return fail(suite, fmt.Sprintf("status code is %d, want 4xx", apiErr.StatusCode))
	}
	if isExcludedErrorStatus(apiErr.StatusCode) {
		return fail(suite, fmt.Sprintf("status code is %d, want client error other than 401/403/429", apiErr.StatusCode))
	}
	errType := apiErr.Type()
	if errType == "" {
		return fail(suite, "error missing type")
	}
	if errType != shared.ErrorTypeInvalidRequestError && errType != shared.ErrorTypeNotFoundError {
		return fail(suite, fmt.Sprintf("error type is %q, want invalid_request_error or not_found_error", errType))
	}
	if !hasModelErrorEvidence(apiErr) {
		return fail(suite, "error lacks model-specific evidence in response body")
	}
	return nil
}

func isExcludedErrorStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func hasModelErrorEvidence(apiErr *anthropic.Error) bool {
	raw := strings.ToLower(apiErr.RawJSON())
	if strings.Contains(raw, errorTriggerModel) {
		return true
	}
	return strings.Contains(raw, "model")
}