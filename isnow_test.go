package isnow_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

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

// The shared conformance corpus, run against this implementation. The corpus is
// the language's contract across implementations; a case that passes here and
// fails in isnow.js is a divergence in one of them, not a difference of opinion.

// corpusDir is the sibling grammar repo's conformance corpus; the suite
// self-skips when the checkout is absent (the up.js model).
const corpusDir = "../isnow/conformance"

type corpusCase struct {
	Name      string    `yaml:"name"`
	Isnow     string    `yaml:"isnow"`
	At        string    `yaml:"at"`
	From      string    `yaml:"from"`
	TZ        string    `yaml:"tz"`
	Holds     *bool     `yaml:"holds"`
	Canonical *string   `yaml:"canonical"`
	Next      *[]string `yaml:"next"`
	Prev      *[]string `yaml:"prev"`
	Error     string    `yaml:"error"`
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(corpusDir, "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Skipf("conformance corpus not present at %s", corpusDir)
	}
	all := make([]corpusCase, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		var doc struct {
			Cases []corpusCase `yaml:"cases"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		all = append(all, doc.Cases...)
	}
	return all
}

func TestConformance(t *testing.T) {
	for _, c := range loadCorpus(t) {
		t.Run(c.Name, func(t *testing.T) { runCase(t, c) })
	}
}

func runCase(t *testing.T, c corpusCase) {
	switch {
	case c.Error != "":
		checkError(t, c)
	case c.Canonical != nil:
		checkCanonical(t, c)
	case c.Holds != nil:
		checkHolds(t, c)
	case c.Next != nil:
		checkDerive(t, c, true)
	case c.Prev != nil:
		checkDerive(t, c, false)
	default:
		t.Fatalf("case %s has no assertion", c.Name)
	}
}

func checkError(t *testing.T, c corpusCase) {
	_, err := isnow.Parse(isnow.PatternText(c.Isnow))
	if got := isnow.Code(err); got != c.Error {
		t.Fatalf("Parse(%q) error = %q, want %q", c.Isnow, got, c.Error)
	}
}

func checkCanonical(t *testing.T, c corpusCase) {
	p, err := isnow.Parse(isnow.PatternText(c.Isnow))
	if err != nil {
		t.Fatalf("Parse(%q): %v", c.Isnow, err)
	}
	if p.Canonical() != *c.Canonical {
		t.Fatalf("Canonical(%q) = %q, want %q", c.Isnow, p.Canonical(), *c.Canonical)
	}
}

func checkHolds(t *testing.T, c corpusCase) {
	p, err := isnow.Parse(isnow.PatternText(c.Isnow))
	if err != nil {
		t.Fatalf("Parse(%q): %v", c.Isnow, err)
	}
	if got := p.Holds(mustTime(t, c.At)); got != *c.Holds {
		t.Fatalf("Holds(%q, %s) = %v, want %v", c.Isnow, c.At, got, *c.Holds)
	}
}

func checkDerive(t *testing.T, c corpusCase, isForward bool) {
	p, err := isnow.Parse(isnow.PatternText(c.Isnow))
	if err != nil {
		t.Fatalf("Parse(%q): %v", c.Isnow, err)
	}
	want := c.Next
	if !isForward {
		want = c.Prev
	}
	got := deriveN(p, mustTime(t, c.From), len(*want), isForward)
	assertInstants(t, c, got, *want)
}

func assertInstants(t *testing.T, c corpusCase, got []time.Time, want []string) {
	if len(got) != len(want) {
		t.Fatalf("derive(%q) got %d occurrences, want %d", c.Isnow, len(got), len(want))
	}
	for i := range want {
		if !got[i].Equal(mustTime(t, want[i])) {
			t.Fatalf("derive(%q)[%d] = %s, want %s", c.Isnow, i, got[i].Format(time.RFC3339), want[i])
		}
	}
}

func deriveN(p isnow.Pattern, from time.Time, n int, isForward bool) []time.Time {
	out := make([]time.Time, 0, n)
	cur := from
	for len(out) < n {
		next, ok := deriveStep(p, cur, isForward)
		if !ok {
			break
		}
		out = append(out, next)
		cur = next
	}
	return out
}

func deriveStep(p isnow.Pattern, from time.Time, isForward bool) (time.Time, bool) {
	if isForward {
		return p.Next(from)
	}
	return p.Prev(from)
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return ts
}

// TestPatternIsImmutableAcrossTheLibraryBoundary pins the promise the public
// alias repeats: a caller may hold one parsed Pattern and share copies without
// synchronising. Evaluation must not mutate the receiver — if it cached
// anything derived, two schedulers sharing a pattern would interfere, and the
// symptom would be a missed fire under load with nothing reproducible after.
func TestPatternIsImmutableAcrossTheLibraryBoundary(t *testing.T) {
	t.Parallel()
	original, err := isnow.Parse("*/*/* M-F 09:00")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	shared := original
	before := original.String()

	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = shared.Next(time.Date(2026, 7, 1+i%28, 0, 0, 0, 0, time.UTC))
		}()
	}
	wg.Wait()

	if original.String() != before {
		t.Fatalf("the original changed under a shared copy: %s -> %s", before, original.String())
	}
}
