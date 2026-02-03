package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/mail"
	"net/textproto"
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

	// Collect attachments from original message
	attachments := collectAttachments(original.Payload)

	// Build the MIME message with attachments
	var mimeMsg bytes.Buffer
	writer := multipart.NewWriter(&mimeMsg)

	// Set content type header (will be added to raw message)
	contentType := fmt.Sprintf("multipart/mixed; boundary=\"%s\"", writer.Boundary())

	// Write text part
	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return fmt.Errorf("creating text part: %w", err)
	}
	if _, err := textPart.Write([]byte(forwardBody)); err != nil {
		return fmt.Errorf("writing text part: %w", err)
	}

	// Fetch and attach each attachment
	for _, att := range attachments {
		if err := addAttachmentToMultipart(ctx, svc, writer, messageID, att); err != nil {
			u.Err().Printf("Warning: failed to attach %s: %v", att.Filename, err)
			continue
		}
	}

	// Close the multipart writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	// Build the complete raw message with headers
	var rawMsg bytes.Buffer
	rawMsg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	rawMsg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	rawMsg.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	rawMsg.WriteString("\r\n")
	rawMsg.Write(mimeMsg.Bytes())

	// Send the message using Gmail API's raw format
	sent, err := svc.Users.Messages.Send("me", &gmail.Message{
		Raw: encodeWeb64(rawMsg.String()),
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("sending forwarded message: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"sent":        sent.Id,
			"threadId":    sent.ThreadId,
			"to":          to,
			"subject":     subject,
			"forwarded":   messageID,
			"attachments": len(attachments),
		})
	}

	u.Out().Successf("Forwarded message %s to %s with %d attachment(s) (sent: %s)",
		messageID, to, len(attachments), sent.Id)
	return nil
}

// addAttachmentToMultipart fetches an attachment and adds it to the multipart writer
func addAttachmentToMultipart(ctx context.Context, svc *gmail.Service, writer *multipart.Writer, messageID string, att attachmentInfo) error {
	// Fetch the attachment data
	attachData, err := svc.Users.Messages.Attachments.Get("me", messageID, att.AttachmentID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("fetching attachment: %w", err)
	}

	// Decode the attachment data (Gmail returns URL-safe base64)
	data, err := decodeBase64URLBytes(attachData.Data)
	if err != nil {
		return fmt.Errorf("decoding attachment data: %w", err)
	}

	// Create attachment part header
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", att.MimeType)
	header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.Filename))
	header.Set("Content-Transfer-Encoding", "base64")

	// Create the part
	part, err := writer.CreatePart(header)
	if err != nil {
		return fmt.Errorf("creating attachment part: %w", err)
	}

	// Write the attachment data as base64
	encoder := base64.NewEncoder(base64.StdEncoding, part)
	if _, err := encoder.Write(data); err != nil {
		encoder.Close()
		return fmt.Errorf("writing attachment data: %w", err)
	}
	encoder.Close()

	return nil
}

// encodeWeb64 encodes a string to URL-safe base64 (web-safe)
func encodeWeb64(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

// quotePrintableEncode encodes text as quoted-printable
func quotePrintableEncode(w io.Writer, text string) error {
	// Simple implementation - just write as-is for now
	// For proper implementation, use mime/quotedprintable package
	_, err := w.Write([]byte(text))
	return err
}

// parseAddressList parses a list of email addresses
func parseAddressList(addrs string) ([]*mail.Address, error) {
	return mail.ParseAddressList(addrs)
}
