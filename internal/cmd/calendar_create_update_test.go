package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

func TestCalendarCreateCmd_RunJSON(t *testing.T) {
	origNew := newCalendarService
	t.Cleanup(func() { newCalendarService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/calendar/v3")
		if r.Method == http.MethodPost && path == "/calendars/cal/events" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "ev1",
				"summary": "Meeting",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := calendar.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newCalendarService = func(context.Context, string) (*calendar.Service, error) { return svc, nil }

	u, err := ui.New(ui.Options{Stdout: os.Stdout, Stderr: os.Stderr, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})

	cmd := &CalendarCreateCmd{}
	out := captureStdout(t, func() {
		if err := runKong(t, cmd, []string{
			"cal",
			"--summary", "Meeting",
			"--from", "2025-01-02T10:00:00Z",
			"--to", "2025-01-02T11:00:00Z",
		}, ctx, &RootFlags{Account: "a@b.com"}); err != nil {
			t.Fatalf("runKong: %v", err)
		}
	})
	if !strings.Contains(out, "\"event\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestCalendarCreateCmd_WithMeetAndAttachments(t *testing.T) {
	origNew := newCalendarService
	t.Cleanup(func() { newCalendarService = origNew })

	var sawConference, sawAttachments bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/calendar/v3")
		if r.Method == http.MethodPost && path == "/calendars/cal/events" {
			var body calendar.Event
			_ = json.NewDecoder(r.Body).Decode(&body)
			sawConference = body.ConferenceData != nil
			sawAttachments = len(body.Attachments) > 0
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "ev2",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := calendar.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newCalendarService = func(context.Context, string) (*calendar.Service, error) { return svc, nil }

	u, err := ui.New(ui.Options{Stdout: os.Stdout, Stderr: os.Stderr, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})

	cmd := &CalendarCreateCmd{}
	if err := runKong(t, cmd, []string{
		"cal",
		"--summary", "Meet",
		"--from", "2025-01-02T10:00:00Z",
		"--to", "2025-01-02T11:00:00Z",
		"--with-meet",
		"--attachment", "https://example.com/file",
	}, ctx, &RootFlags{Account: "a@b.com"}); err != nil {
		t.Fatalf("runKong: %v", err)
	}
	if !sawConference || !sawAttachments {
		t.Fatalf("expected conference+attachments, sawConference=%v sawAttachments=%v", sawConference, sawAttachments)
	}
}

func TestCalendarUpdateCmd_RunJSON(t *testing.T) {
	origNew := newCalendarService
	t.Cleanup(func() { newCalendarService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/calendar/v3")
		if r.Method == http.MethodPatch && path == "/calendars/cal/events/ev" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "ev",
				"summary": "Updated",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := calendar.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newCalendarService = func(context.Context, string) (*calendar.Service, error) { return svc, nil }

	u, err := ui.New(ui.Options{Stdout: os.Stdout, Stderr: os.Stderr, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})

	cmd := &CalendarUpdateCmd{}
	out := captureStdout(t, func() {
		if err := runKong(t, cmd, []string{
			"cal",
			"ev",
			"--summary", "Updated",
			"--scope", "all",
		}, ctx, &RootFlags{Account: "a@b.com"}); err != nil {
			t.Fatalf("runKong: %v", err)
		}
	})
	if !strings.Contains(out, "\"event\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestCalendarUpdateCmd_AddAttendee(t *testing.T) {
	origNew := newCalendarService
	t.Cleanup(func() { newCalendarService = origNew })

	var patchedAttendees int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/calendar/v3")
		switch {
		case r.Method == http.MethodGet && path == "/calendars/cal/events/ev":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "ev",
				"attendees": []map[string]any{
					{"email": "a@example.com"},
				},
			})
			return
		case r.Method == http.MethodPatch && path == "/calendars/cal/events/ev":
			var body calendar.Event
			_ = json.NewDecoder(r.Body).Decode(&body)
			patchedAttendees = len(body.Attendees)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "ev",
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	defer srv.Close()

	svc, err := calendar.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newCalendarService = func(context.Context, string) (*calendar.Service, error) { return svc, nil }

	u, err := ui.New(ui.Options{Stdout: os.Stdout, Stderr: os.Stderr, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})

	cmd := &CalendarUpdateCmd{}
	if err := runKong(t, cmd, []string{
		"cal",
		"ev",
		"--add-attendee", "b@example.com",
		"--scope", "all",
	}, ctx, &RootFlags{Account: "a@b.com"}); err != nil {
		t.Fatalf("runKong: %v", err)
	}
	if patchedAttendees < 2 {
		t.Fatalf("expected merged attendees, got %d", patchedAttendees)
	}
}

func TestCalendarUpdateCmd_ScopeFuture(t *testing.T) {
	origNew := newCalendarService
	t.Cleanup(func() { newCalendarService = origNew })

	var truncated bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/calendar/v3")
		switch {
		case r.Method == http.MethodGet && path == "/calendars/cal/events/ev":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         "ev",
				"recurrence": []string{"RRULE:FREQ=DAILY"},
			})
			return
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/calendars/cal/events/ev/instances"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": "ev_1",
						"originalStartTime": map[string]any{
							"dateTime": "2025-01-02T10:00:00Z",
						},
					},
				},
			})
			return
		case r.Method == http.MethodPatch && path == "/calendars/cal/events/ev_1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "ev_1"})
			return
		case r.Method == http.MethodPatch && path == "/calendars/cal/events/ev":
			truncated = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "ev"})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	defer srv.Close()

	svc, err := calendar.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newCalendarService = func(context.Context, string) (*calendar.Service, error) { return svc, nil }

	u, err := ui.New(ui.Options{Stdout: os.Stdout, Stderr: os.Stderr, Color: "never"})
	if err != nil {
		t.Fatalf("ui.New: %v", err)
	}
	ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{JSON: true})

	cmd := &CalendarUpdateCmd{}
	if err := runKong(t, cmd, []string{
		"cal",
		"ev",
		"--summary", "Updated",
		"--scope", "future",
		"--original-start", "2025-01-02T10:00:00Z",
	}, ctx, &RootFlags{Account: "a@b.com"}); err != nil {
		t.Fatalf("runKong: %v", err)
	}
	if !truncated {
		t.Fatalf("expected recurrence truncation")
	}
}
