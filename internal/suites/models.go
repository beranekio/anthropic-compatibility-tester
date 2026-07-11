package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// Models verifies GET /v1/models via client.Models.List.
type Models struct{}

func (Models) Name() string        { return "models" }
func (Models) Description() string { return "List models (GET /v1/models)" }

func (Models) Run(ctx context.Context, client anthropic.Client, _ *config.Config) error {
	page, err := client.Models.List(ctx, anthropic.ModelListParams{})
	if err != nil {
		return fmt.Errorf("models list request failed: %w", err)
	}
	if page == nil {
		return fail("models", "response is nil")
	}
	if len(page.Data) == 0 {
		return fail("models", "expected at least one model in list response")
	}
	for _, model := range page.Data {
		if model.ID == "" {
			return fail("models", "model entry missing id")
		}
		if model.DisplayName == "" {
			return fail("models", "model entry missing display_name")
		}
		if string(model.Type) != "model" {
			return fail("models", fmt.Sprintf("model type is %q, want model", model.Type))
		}
	}
	return nil
}