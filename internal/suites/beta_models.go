package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// BetaModels verifies GET /v1/models?beta=true via client.Beta.Models.List.
type BetaModels struct{}

func (BetaModels) Name() string { return "beta_models" }
func (BetaModels) Description() string {
	return "Beta list models (GET /v1/models?beta=true)"
}

func (BetaModels) Run(ctx context.Context, client anthropic.Client, _ *config.Config) error {
	page, err := client.Beta.Models.List(ctx, anthropic.BetaModelListParams{})
	if err != nil {
		return fmt.Errorf("beta models list request failed: %w", err)
	}
	if page == nil {
		return fail("beta_models", "response is nil")
	}
	if len(page.Data) == 0 {
		return fail("beta_models", "expected at least one model in list response")
	}
	for _, model := range page.Data {
		if model.ID == "" {
			return fail("beta_models", "model entry missing id")
		}
		if model.DisplayName == "" {
			return fail("beta_models", "model entry missing display_name")
		}
		if string(model.Type) != "model" {
			return fail("beta_models", fmt.Sprintf("model type is %q, want model", model.Type))
		}
	}
	return nil
}
