package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/gmail/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type GmailMessagesForwardCmd struct {
	MessageID string `arg:"" name:"messageId" help:"Message ID to forward"`
	To        string `name:"to" required:"" help:"Recipient email address"`
	Subject   string `name:"subject" help:"Optional subject (default: Fwd: original subject)"`
}

func (c *GmailMessagesForwardCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	messageID := strings.TrimSpace(c.MessageID)
	if messageID == "" {
		return usage("message ID is required")
	}

	to := strings.TrimSpace(c.To)
	if to == "" {
		return usage("--to is required")
	}

	svc, err := newGmailService(ctx, account)
	if err != nil {
		return err
	}

	// Fetch the original message
	original, err := svc.Users.Messages.Get("me", messageID).Format("full").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("fetching message: %w", err)
	}

	// Get original headers
	originalSubject := headerValue(original.Payload, "Subject")
	originalFrom := headerValue(original.Payload, "From")
	originalTo := headerValue(original.Payload, "To")
	originalDate := headerValue(original.Payload, "Date")

	// Build subject
	subject := c.Subject
	if subject == "" {
		subject = "Fwd: " + originalSubject
	}

	// Build the forwarded message body
	originalBody, _ := bestBodyForDisplay(original.Payload)
	forwardBody := fmt.Sprintf("---------- Forwarded message ----------\nFrom: %s\nDate: %s\nSubject: %s\nTo: %s\n\n%s",
		originalFrom, originalDate, originalSubject, originalTo, originalBody)

	// Build raw RFC2822 message
	var rawMsg strings.Builder
	rawMsg.WriteString(fmt.Sprintf("To: %s\n", to))
	rawMsg.WriteString(fmt.Sprintf("Subject: %s\n", subject))
	rawMsg.WriteString("Content-Type: text/plain; charset=utf-8\n")
	rawMsg.WriteString("\n")
	rawMsg.WriteString(forwardBody)

	// Send the message using Gmail API's raw format
	sent, err := svc.Users.Messages.Send("me", &gmail.Message{
		Raw: encodeWeb64(rawMsg.String()),
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("sending forwarded message: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"sent":      sent.Id,
			"threadId":  sent.ThreadId,
			"to":        to,
			"subject":   subject,
			"forwarded": messageID,
		})
	}

	u.Out().Successf("Forwarded message %s to %s (sent: %s)", messageID, to, sent.Id)
	return nil
}

// encodeWeb64 encodes a string to URL-safe base64 (web-safe)
func encodeWeb64(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}
