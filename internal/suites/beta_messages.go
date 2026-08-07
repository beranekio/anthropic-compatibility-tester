package suites

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
)

// BetaMessages verifies POST /v1/messages?beta=true via client.Beta.Messages.New.
type BetaMessages struct{}

func (BetaMessages) Name() string { return "beta_messages" }
func (BetaMessages) Description() string {
	return "Beta message completion (POST /v1/messages?beta=true)"
}

func (BetaMessages) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	msg, err := client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(cfg.Model),
		MaxTokens: 64,
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Reply with exactly the word: pong")),
		},
	})
	if err != nil {
		return fmt.Errorf("beta messages request failed: %w", err)
	}
	if err := validateBetaMessageEnvelope("beta_messages", msg); err != nil {
		return err
	}
	return validateBetaMessageHasTextOutput("beta_messages", msg)
}

func validateBetaMessageEnvelope(suite string, msg *anthropic.BetaMessage) error {
	if msg == nil {
		return fail(suite, "response is nil")
	}
	if msg.ID == "" {
		return fail(suite, "response missing id")
	}
	if msg.Model == "" {
		return fail(suite, "response missing model")
	}
	if string(msg.Type) != "message" {
		return fail(suite, fmt.Sprintf("response type is %q, want message", msg.Type))
	}
	if string(msg.Role) != "assistant" {
		return fail(suite, fmt.Sprintf("response role is %q, want assistant", msg.Role))
	}
	if msg.StopReason == "" {
		return fail(suite, "response missing stop_reason")
	}
	return nil
}

func validateBetaMessageHasTextOutput(suite string, msg *anthropic.BetaMessage) error {
	if msg == nil {
		return fail(suite, "response is nil")
	}
	if string(msg.StopReason) == "refusal" {
		return nil
	}
	if betaMessageTextContent(msg) != "" {
		return nil
	}
	return fail(suite, "response produced no text content")
}

func betaMessageTextContent(msg *anthropic.BetaMessage) string {
	if msg == nil {
		return ""
	}
	var b strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" && block.Text != "" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}
