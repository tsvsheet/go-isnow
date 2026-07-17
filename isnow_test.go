package isnow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	isnow "github.com/tsvsheet/go-isnow"
)

// TestFacadeSurface exercises every public wrapper the facade adds over
// internal/engine: a caller parses, tests membership, derives occurrences, and
// classifies errors without ever naming the internal engine.
func TestFacadeSurface(t *testing.T) {
	t.Parallel()
	p, err := isnow.Parse("M noon")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	monday := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	if !p.Holds(monday) {
		t.Fatalf("Holds(%s) = false, want true", monday)
	}
	if got := p.Canonical(); got != p.String() {
		t.Fatalf("String() = %q, want Canonical() %q", p.String(), got)
	}
	if p.Explain() == "" {
		t.Fatal("Explain() is empty")
	}
	ok, err := isnow.Is("M noon", monday)
	if err != nil || !ok {
		t.Fatalf("Is = (%v, %v), want (true, nil)", ok, err)
	}
	if _, err := isnow.Is("///", monday); !errors.Is(err, isnow.ErrContext) {
		t.Fatalf("Is on invalid pattern = %v, want ErrContext", err)
	}
}

// TestFacadeErrors asserts each sentinel surfaces through the facade and maps
// to its stable cross-implementation code.
func TestFacadeErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src  isnow.PatternText
		want error
		code string
	}{
		{"5-", isnow.ErrSyntax, "syntax"},
		{"xyzzy", isnow.ErrSymbol, "symbol"},
		{"25:00", isnow.ErrRange, "range"},
		{"/// noon", isnow.ErrContext, "context"},
	}
	for _, c := range cases {
		_, err := isnow.Parse(c.src)
		if !errors.Is(err, c.want) {
			t.Fatalf("Parse(%q) = %v, want %v", c.src, err, c.want)
		}
		if got := isnow.Code(err); got != c.code {
			t.Fatalf("Code(Parse(%q)) = %q, want %q", c.src, got, c.code)
		}
	}
	if got := isnow.Code(nil); got != "" {
		t.Fatalf("Code(nil) = %q, want \"\"", got)
	}
}

// TestDeriveContext pins the cancellation contract: a live context derives
// exactly what Next/Prev derive, and a cancelled context aborts with its error.
func TestDeriveContext(t *testing.T) {
	t.Parallel()
	p, err := isnow.Parse("M noon")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	from := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	next, ok, err := p.NextContext(context.Background(), from)
	if err != nil || !ok {
		t.Fatalf("NextContext = (%v, %v, %v), want an occurrence", next, ok, err)
	}
	if want, _ := p.Next(from); !next.Equal(want) {
		t.Fatalf("NextContext = %s, want Next's %s", next, want)
	}
	prev, ok, err := p.PrevContext(context.Background(), from)
	if err != nil || !ok {
		t.Fatalf("PrevContext = (%v, %v, %v), want an occurrence", prev, ok, err)
	}
	if want, _ := p.Prev(from); !prev.Equal(want) {
		t.Fatalf("PrevContext = %s, want Prev's %s", prev, want)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok, err := p.NextContext(cancelled, from); ok || !errors.Is(err, context.Canceled) {
		t.Fatalf("NextContext(cancelled) = (ok=%v, %v), want (false, context.Canceled)", ok, err)
	}
	if _, ok, err := p.PrevContext(cancelled, from); ok || !errors.Is(err, context.Canceled) {
		t.Fatalf("PrevContext(cancelled) = (ok=%v, %v), want (false, context.Canceled)", ok, err)
	}
}
