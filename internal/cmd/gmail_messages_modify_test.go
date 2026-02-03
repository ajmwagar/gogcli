package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

func TestGmailMessagesModifyCmd_JSON(t *testing.T) {
	origNew := newGmailService
	t.Cleanup(func() { newGmailService = origNew })

	var receivedRequest *gmail.BatchModifyMessagesRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/users/me/labels"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"labels": []map[string]any{
					{"id": "INBOX", "name": "INBOX", "type": "system"},
					{"id": "Label_1", "name": "Custom", "type": "user"},
					{"id": "Label_2", "name": "Work", "type": "user"},
				},
			})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/users/me/messages/batchModify"):
			receivedRequest = &gmail.BatchModifyMessagesRequest{}
			_ = json.NewDecoder(r.Body).Decode(receivedRequest)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
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

		if err := runKong(t, &GmailMessagesModifyCmd{}, []string{
			"--ids", "m1,m2",
			"--add-label", "Work",
			"--remove-label", "Custom",
		}, ctx, flags); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if receivedRequest == nil {
		t.Fatal("no request received")
	}
	if len(receivedRequest.Ids) != 2 || receivedRequest.Ids[0] != "m1" || receivedRequest.Ids[1] != "m2" {
		t.Fatalf("unexpected ids: %#v", receivedRequest.Ids)
	}
	if len(receivedRequest.AddLabelIds) != 1 || receivedRequest.AddLabelIds[0] != "Label_2" {
		t.Fatalf("unexpected add labels: %#v", receivedRequest.AddLabelIds)
	}
	if len(receivedRequest.RemoveLabelIds) != 1 || receivedRequest.RemoveLabelIds[0] != "Label_1" {
		t.Fatalf("unexpected remove labels: %#v", receivedRequest.RemoveLabelIds)
	}

	var parsed struct {
		Modified      []string `json:"modified"`
		Count         int      `json:"count"`
		AddedLabels   []string `json:"addedLabels"`
		RemovedLabels []string `json:"removedLabels"`
		Archived      bool     `json:"archived"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if parsed.Count != 2 {
		t.Fatalf("unexpected count: %d", parsed.Count)
	}
	if len(parsed.AddedLabels) != 1 || parsed.AddedLabels[0] != "Label_2" {
		t.Fatalf("unexpected added labels: %#v", parsed.AddedLabels)
	}
	if parsed.Archived {
		t.Fatalf("unexpected archived=true")
	}
}

func TestGmailMessagesModifyCmd_Archive(t *testing.T) {
	origNew := newGmailService
	t.Cleanup(func() { newGmailService = origNew })

	var receivedRequest *gmail.BatchModifyMessagesRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/users/me/labels"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"labels": []map[string]any{
					{"id": "INBOX", "name": "INBOX", "type": "system"},
				},
			})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/users/me/messages/batchModify"):
			receivedRequest = &gmail.BatchModifyMessagesRequest{}
			_ = json.NewDecoder(r.Body).Decode(receivedRequest)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
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

		if err := runKong(t, &GmailMessagesModifyCmd{}, []string{
			"--ids", "m1",
			"--archive",
		}, ctx, flags); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if receivedRequest == nil {
		t.Fatal("no request received")
	}
	if len(receivedRequest.Ids) != 1 || receivedRequest.Ids[0] != "m1" {
		t.Fatalf("unexpected ids: %#v", receivedRequest.Ids)
	}
	if len(receivedRequest.RemoveLabelIds) != 1 || receivedRequest.RemoveLabelIds[0] != "INBOX" {
		t.Fatalf("expected INBOX in remove labels for archive: %#v", receivedRequest.RemoveLabelIds)
	}

	var parsed struct {
		Archived bool `json:"archived"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if !parsed.Archived {
		t.Fatal("expected archived=true in output")
	}
}

func TestGmailMessagesModifyCmd_ArchiveNoDuplicate(t *testing.T) {
	origNew := newGmailService
	t.Cleanup(func() { newGmailService = origNew })

	var receivedRequest *gmail.BatchModifyMessagesRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/users/me/labels"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"labels": []map[string]any{
					{"id": "INBOX", "name": "INBOX", "type": "system"},
				},
			})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/users/me/messages/batchModify"):
			receivedRequest = &gmail.BatchModifyMessagesRequest{}
			_ = json.NewDecoder(r.Body).Decode(receivedRequest)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
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

	captureStdout(t, func() {
		u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
		if uiErr != nil {
			t.Fatalf("ui.New: %v", uiErr)
		}
		ctx := ui.WithUI(context.Background(), u)
		ctx = outfmt.WithMode(ctx, outfmt.Mode{JSON: true})

		// Specify both --archive and --remove-label INBOX
		if err := runKong(t, &GmailMessagesModifyCmd{}, []string{
			"--ids", "m1",
			"--archive",
			"--remove-label", "INBOX",
		}, ctx, flags); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if receivedRequest == nil {
		t.Fatal("no request received")
	}
	// Should only have INBOX once, not duplicated
	if len(receivedRequest.RemoveLabelIds) != 1 {
		t.Fatalf("expected single INBOX in remove labels, got: %#v", receivedRequest.RemoveLabelIds)
	}
	if receivedRequest.RemoveLabelIds[0] != "INBOX" {
		t.Fatalf("expected INBOX in remove labels: %#v", receivedRequest.RemoveLabelIds)
	}
}

func TestGmailMessagesModifyCmd_PlainText(t *testing.T) {
	origNew := newGmailService
	t.Cleanup(func() { newGmailService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/users/me/labels"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"labels": []map[string]any{
					{"id": "INBOX", "name": "INBOX", "type": "system"},
				},
			})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/users/me/messages/batchModify"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
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
		u, uiErr := ui.New(ui.Options{Stdout: os.Stdout, Stderr: io.Discard, Color: "never"})
		if uiErr != nil {
			t.Fatalf("ui.New: %v", uiErr)
		}
		ctx := ui.WithUI(context.Background(), u)

		if err := runKong(t, &GmailMessagesModifyCmd{}, []string{
			"--ids", "m1,m2,m3",
			"--archive",
		}, ctx, flags); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if !strings.Contains(out, "Modified 3 message(s)") {
		t.Fatalf("unexpected plain output: %q", out)
	}
}

func TestGmailMessagesModifyCmd_NoActionError(t *testing.T) {
	origNew := newGmailService
	t.Cleanup(func() { newGmailService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
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

	u, uiErr := ui.New(ui.Options{Stdout: io.Discard, Stderr: io.Discard, Color: "never"})
	if uiErr != nil {
		t.Fatalf("ui.New: %v", uiErr)
	}
	ctx := ui.WithUI(context.Background(), u)

	err = runKong(t, &GmailMessagesModifyCmd{}, []string{
		"--ids", "m1",
	}, ctx, flags)

	if err == nil {
		t.Fatal("expected error when no action specified")
	}
	if !strings.Contains(err.Error(), "must specify") {
		t.Fatalf("unexpected error: %v", err)
	}
}
