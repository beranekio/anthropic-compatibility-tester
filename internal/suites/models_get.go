package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// ModelsGet verifies GET /v1/models/{id} via client.Models.Get.
type ModelsGet struct{}

func (ModelsGet) Name() string        { return "models_get" }
func (ModelsGet) Description() string { return "Retrieve model by ID (GET /v1/models/{id})" }

func (ModelsGet) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	model, err := client.Models.Get(ctx, cfg.Model, anthropic.ModelGetParams{})
	if err != nil {
		return fmt.Errorf("models get request failed: %w", err)
	}
	if model == nil {
		return fail("models_get", "response is nil")
	}
	if model.ID == "" {
		return fail("models_get", "model missing id")
	}
	if model.ID != cfg.Model {
		return fail("models_get", fmt.Sprintf("model id is %q, want %q", model.ID, cfg.Model))
	}
	if string(model.Type) != "model" {
		return fail("models_get", fmt.Sprintf("model type is %q, want model", model.Type))
	}
	if model.DisplayName == "" {
		return fail("models_get", "model missing display_name")
	}
	return nil
}