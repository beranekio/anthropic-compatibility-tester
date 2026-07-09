package suites

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// smallPNGDataURL is an 8x8 PNG encoded as a data URL for vision requests.
const smallPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAgAAAAICAYAAADED76LAAAAEklEQVR4nGP4n2L0Hx9mGBkKACBDpQFoN/xgAAAAAElFTkSuQmCC"

// MessagesVision verifies multimodal POST /v1/messages with image input.
type MessagesVision struct{}

func (MessagesVision) Name() string { return "messages_vision" }
func (MessagesVision) Description() string {
	return "Message completion with vision input (POST /v1/messages)"
}

func (MessagesVision) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.VisionModel),
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock("Describe this image in one short sentence."),
				anthropic.NewImageBlockBase64("image/png", smallPNGDataURL[len("data:image/png;base64,"):]),
			),
		},
	})
	if err != nil {
		return fmt.Errorf("messages vision request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_vision", msg); err != nil {
		return err
	}
	return validateMessageHasTextOutput("messages_vision", msg)
}