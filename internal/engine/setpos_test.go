package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tsvsheet/go-isnow/internal/constants"
)

func TestSetposLastBusinessDay(t *testing.T) {
	p := mustParse(t, "M-F-[1] noon")
	if p.Canonical() != "*/*/* Monday-Friday-[1] 12:00:00" {
		t.Fatalf("Canonical = %q", p.Canonical())
	}
	// Feb 2026: last day is Sat 28; last business day is Fri 27.
	cases := []struct {
		day  string
		want bool
	}{
		{"2026-02-27T12:00:00Z", true},  // Friday, last business day
		{"2026-02-26T12:00:00Z", false}, // Thursday
		{"2026-02-28T12:00:00Z", false}, // Saturday
	}
	for _, c := range cases {
		if got := p.Holds(at(t, c.day)); got != c.want {
			t.Fatalf("Holds(%s) = %v, want %v", c.day, got, c.want)
		}
	}
}

func TestSetposFirstBusinessDayAndNumericSpan(t *testing.T) {
	// Mar 2026 starts on Sun; first business day is Mon Mar 2.
	if !mustParse(t, "M-F+[1] noon").Holds(at(t, "2026-03-02T12:00:00Z")) {
		t.Fatal("first business day should hold Mon Mar 2")
	}
	// A numeric-value span endpoint also resolves as a weekday (2-6 = Mon-Fri).
	if !mustParse(t, "*/*/* 2-6-[1] 12:00").Holds(at(t, "2026-02-27T12:00:00Z")) {
		t.Fatal("numeric weekday span BYSETPOS should hold")
	}
}

func TestSetposDerivation(t *testing.T) {
	p := mustParse(t, "M-F-[1] noon")
	got, ok := p.Next(at(t, "2026-01-01T00:00:00Z"))
	if !ok || !got.Equal(at(t, "2026-01-30T12:00:00Z")) {
		t.Fatalf("Next = %s, %v", got.Format(time.RFC3339), ok)
	}
}

func TestSetposIndexOutOfRange(t *testing.T) {
	if _, err := Parse("M-F+[99] noon"); !errors.Is(err, constants.ErrRange) {
		t.Fatalf("index 99 should be range: %v", err)
	}
	if _, err := Parse("M-F+[0] noon"); !errors.Is(err, constants.ErrRange) {
		t.Fatalf("index 0 should be range: %v", err)
	}
}

func TestSetposBadEndpoints(t *testing.T) {
	// A run (MWF) passes bare-group classification but is not a single span
	// endpoint, so BYSETPOS rejects it at the low endpoint.
	if _, err := Parse("MWF-F-[1] noon"); !errors.Is(err, constants.ErrSymbol) {
		t.Fatalf("run as low endpoint: %v", err)
	}
	if _, err := Parse("*/*/* 2-9-[1] 12:00"); !errors.Is(err, constants.ErrRange) {
		t.Fatalf("out-of-range numeric endpoint: %v", err)
	}
	// A wildcard endpoint is not a concrete weekday.
	if _, err := Parse("*/*/* *-M+[1] 12:00"); !errors.Is(err, constants.ErrContext) {
		t.Fatalf("wildcard BYSETPOS endpoint: %v", err)
	}
}

func TestFuzzRegressions(t *testing.T) {
	// Weekday spans need concrete endpoints; open-high is a context error.
	if _, err := Parse("*/*/* 1-* 12:00"); !errors.Is(err, constants.ErrContext) {
		t.Fatalf("weekday v-* should be context: %v", err)
	}
	// An overflow member in a from-end unit compound invalidates the tail.
	if _, err := Parse("*/*/-10000000000000000000d2 *:*:00"); !errors.Is(err, constants.ErrRange) {
		t.Fatalf("overflow compound should be range: %v", err)
	}
}

// TestMonthRankCountsFromWhicheverEdgeWasNamed pins the counting direction.
// "the second Tuesday" and "the second-to-last Tuesday" are different days in
// most months and the same day in some, so a rank counted from the wrong edge
// is wrong in a way that looks right roughly half the time — the worst
// signature a scheduling bug can have.
func TestMonthRankCountsFromWhicheverEdgeWasNamed(t *testing.T) {
	t.Parallel()
	// July 2026: Wednesdays fall on the 1st, 8th, 15th, 22nd and 29th.
	second, err := Parse("*/*/* W+[2] 12:00")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	fromEnd, err := Parse("*/*/* W-[1] 12:00")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	gotSecond, ok, _ := second.NextContext(context.Background(), start)
	if !ok {
		t.Fatal("no match for the second Wednesday")
	}
	gotLast, ok, _ := fromEnd.NextContext(context.Background(), start)
	if !ok {
		t.Fatal("no match for the last Wednesday")
	}

	if gotSecond.Day() == gotLast.Day() {
		t.Fatalf("counting from each edge gave the same day (%d); the direction is not being honoured", gotSecond.Day())
	}
}
