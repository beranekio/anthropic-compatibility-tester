package suites

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
	"github.com/beranekio/anthropic-compatibility-tester/internal/testutil"
)

// MessagesDocument verifies POST /v1/messages with a document/PDF content block.
type MessagesDocument struct{}

func (MessagesDocument) Name() string { return "messages_document" }
func (MessagesDocument) Description() string {
	return "Message completion with PDF document input (POST /v1/messages)"
}

func (MessagesDocument) Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error {
	pdfB64 := base64.StdEncoding.EncodeToString(testutil.MinimalPDFBytes())
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.VisionModel),
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock("Summarize this document in one short sentence."),
				anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{
					Data: pdfB64,
				}),
			),
		},
	})
	if err != nil {
		return fmt.Errorf("messages document request failed: %w", err)
	}
	if err := validateMessageEnvelope("messages_document", msg); err != nil {
		return err
	}
	return validateMessageHasTextOutput("messages_document", msg)
}
