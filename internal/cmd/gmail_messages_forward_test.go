package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

func TestGmailMessagesForwardCmd_JSON(t *testing.T) {
	origNew := newGmailService
	t.Cleanup(func() { newGmailService = origNew })

	var receivedRequest *gmail.Message

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/users/me/messages/m1"):
			// Return original message
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "m1",
				"threadId": "t1",
				"labelIds": []string{"INBOX"},
				"payload": map[string]any{
					"headers": []map[string]string{
						{"name": "From", "value": "sender@example.com"},
						{"name": "To", "value": "me@example.com"},
						{"name": "Subject", "value": "Original Subject"},
						{"name": "Date", "value": "Mon, 03 Feb 2026 10:00:00 -0800"},
					},
					"body": map[string]any{
						"data": "SGVsbG8gV29ybGQ=", // "Hello World" in base64
					},
				},
			})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/users/me/messages/send"):
			// Capture sent message
			receivedRequest = &gmail.Message{}
			_ = json.NewDecoder(r.Body).Decode(receivedRequest)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "sent123",
				"threadId": "t1",
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	defer srv.Close()

	svc, err := gmail.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newGmailService = func(context.Context, string) (*gmail.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "a@b.com"}

	out := captureStdout(t, func() {
		u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
		if uiErr != nil {
			t.Fatalf("ui.New: %v", uiErr)
		}
		ctx := ui.WithUI(context.Background(), u)
		ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

		if err := runKong(t, &GmailMessagesForwardCmd{}, []string{
			"m1",
			"--to", "recipient@example.com",
		}, ctx, flags); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if receivedRequest == nil {
		t.Fatal("no send request received")
	}

	var parsed struct {
		Sent      string `json:"sent"`
		ThreadID  string `json:"threadId"`
		To        string `json:"to"`
		Subject   string `json:"subject"`
		Forwarded string `json:"forwarded"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if parsed.Sent != "sent123" {
		t.Fatalf("unexpected sent ID: %s", parsed.Sent)
	}
	if parsed.To != "recipient@example.com" {
		t.Fatalf("unexpected to: %s", parsed.To)
	}
	if !strings.Contains(parsed.Subject, "Fwd:") {
		t.Fatalf("expected Fwd: in subject: %s", parsed.Subject)
	}
	if parsed.Forwarded != "m1" {
		t.Fatalf("unexpected forwarded ID: %s", parsed.Forwarded)
	}
}

func TestGmailMessagesForwardCmd_CustomSubject(t *testing.T) {
	origNew := newGmailService
	t.Cleanup(func() { newGmailService = origNew })

	var receivedRequest *gmail.Message

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/users/me/messages/m1"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "m1",
				"threadId": "t1",
				"payload": map[string]any{
					"headers": []map[string]string{
						{"name": "From", "value": "sender@example.com"},
						{"name": "Subject", "value": "Original"},
						{"name": "Date", "value": "Mon, 03 Feb 2026 10:00:00 -0800"},
					},
					"body": map[string]any{"data": ""},
				},
			})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/users/me/messages/send"):
			receivedRequest = &gmail.Message{}
			_ = json.NewDecoder(r.Body).Decode(receivedRequest)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "sent123", "threadId": "t1"})
			return
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc, err := gmail.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newGmailService = func(context.Context, string) (*gmail.Service, error) { return svc, nil }

	flags := &RootFlags{Account: "a@b.com"}

	out := captureStdout(t, func() {
		u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
		if uiErr != nil {
			t.Fatalf("ui.New: %v", uiErr)
		}
		ctx := ui.WithUI(context.Background(), u)
		ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

		if err := runKong(t, &GmailMessagesForwardCmd{}, []string{
			"m1",
			"--to", "recipient@example.com",
			"--subject", "Custom Forward Subject",
		}, ctx, flags); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	var parsed struct {
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if parsed.Subject != "Custom Forward Subject" {
		t.Fatalf("expected custom subject, got: %s", parsed.Subject)
	}
}

func TestGmailMessagesForwardCmd_MissingTo(t *testing.T) {
	flags := &RootFlags{Account: "a@b.com"}

	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)

	err := runKong(t, &GmailMessagesForwardCmd{}, []string{
		"m1",
	}, ctx, flags)

	if err == nil {
		t.Fatal("expected error when --to is missing")
	}
}
