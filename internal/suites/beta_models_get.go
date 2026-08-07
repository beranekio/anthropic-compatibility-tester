package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// BetaModelsGet verifies GET /v1/models/{id}?beta=true via client.Beta.Models.Get.
type BetaModelsGet struct{}

func (BetaModelsGet) Name() string { return "beta_models_get" }
func (BetaModelsGet) Description() string {
	return "Beta retrieve model by ID (GET /v1/models/{id}?beta=true)"
}

func (BetaModelsGet) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	model, err := client.Beta.Models.Get(ctx, cfg.Model, anthropic.BetaModelGetParams{})
	if err != nil {
		return fmt.Errorf("beta models get request failed: %w", err)
	}
	if model == nil {
		return fail("beta_models_get", "response is nil")
	}
	if model.ID == "" {
		return fail("beta_models_get", "model missing id")
	}
	if model.ID != cfg.Model {
		return fail("beta_models_get", fmt.Sprintf("model id is %q, want %q", model.ID, cfg.Model))
	}
	if string(model.Type) != "model" {
		return fail("beta_models_get", fmt.Sprintf("model type is %q, want model", model.Type))
	}
	if model.DisplayName == "" {
		return fail("beta_models_get", "model missing display_name")
	}
	return nil
}
