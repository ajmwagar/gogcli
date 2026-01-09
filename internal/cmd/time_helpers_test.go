package cmd

import (
	"testing"
	"time"
)

func TestParseTimeExpr(t *testing.T) {
	now := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	parsed, err := parseTimeExpr("today", now, time.UTC)
	if err != nil {
		t.Fatalf("parseTimeExpr today: %v", err)
	}
	if !parsed.Equal(startOfDay(now)) {
		t.Fatalf("unexpected today: %v", parsed)
	}

	parsed, err = parseTimeExpr("2025-01-05", now, time.UTC)
	if err != nil {
		t.Fatalf("parseTimeExpr date: %v", err)
	}
	if parsed.Year() != 2025 || parsed.Day() != 5 {
		t.Fatalf("unexpected date: %v", parsed)
	}

	if _, err = parseTimeExpr("nope", now, time.UTC); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestParseWeekday(t *testing.T) {
	now := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	parsed, ok := parseWeekday("monday", now)
	if !ok || parsed.Weekday() != time.Monday {
		t.Fatalf("unexpected weekday: %v ok=%v", parsed, ok)
	}
}

func TestResolveWeekStart(t *testing.T) {
	day, err := resolveWeekStart("sun")
	if err != nil || day != time.Sunday {
		t.Fatalf("unexpected week start: %v %v", day, err)
	}
	if _, err = resolveWeekStart("nope"); err == nil {
		t.Fatalf("expected error for invalid week start")
	}
}

func TestTimeRangeFormatting(t *testing.T) {
	tr := &TimeRange{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	from, to := tr.FormatRFC3339()
	if from == "" || to == "" {
		t.Fatalf("expected formatted range")
	}
	if tr.FormatHuman() == "" {
		t.Fatalf("expected human format")
	}
}

func TestWeekBounds(t *testing.T) {
	now := time.Date(2025, 1, 8, 12, 0, 0, 0, time.UTC) // Wednesday
	start := startOfWeek(now, time.Monday)
	end := endOfWeek(now, time.Monday)
	if start.Weekday() != time.Monday || end.Weekday() != time.Sunday {
		t.Fatalf("unexpected week bounds: %v to %v", start.Weekday(), end.Weekday())
	}

	startSun := startOfWeek(now, time.Sunday)
	endSun := endOfWeek(now, time.Sunday)
	if startSun.Weekday() != time.Sunday || endSun.Weekday() != time.Saturday {
		t.Fatalf("unexpected week bounds (sun): %v to %v", startSun.Weekday(), endSun.Weekday())
	}
}

func TestDayBounds(t *testing.T) {
	now := time.Date(2025, 1, 8, 12, 34, 56, 0, time.UTC)
	start := startOfDay(now)
	end := endOfDay(now)
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Fatalf("unexpected startOfDay: %v", start)
	}
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Fatalf("unexpected endOfDay: %v", end)
	}
}

func TestParseWeekStartVariants(t *testing.T) {
	if wd, ok := parseWeekStart("tues"); !ok || wd != time.Tuesday {
		t.Fatalf("unexpected week start: %v ok=%v", wd, ok)
	}
	if _, ok := parseWeekStart("nope"); ok {
		t.Fatalf("expected invalid week start")
	}
}
