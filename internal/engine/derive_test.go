package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNextContextCannotPinACPUIndefinitely pins the reason NextContext exists.
// A pattern can describe an instant that never arrives — an impossible bounded
// window — and Next would scan forward forever looking for it. A library that
// can be made to spin by its own input is a denial of service in whatever
// process embeds it, so the context is checked once per civil day: cancellation
// is observed within a single day's enumeration rather than never.
func TestNextContextCannotPinACPUIndefinitely(t *testing.T) {
	t.Parallel()
	// February 30th parses and never arrives, so the scan has no stopping
	// point: exactly the input this guard exists for.
	impossible, err := Parse("*/2/30 12:00")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, err = impossible.NextContext(ctx, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	}()

	select {
	case <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("NextContext ignored a cancelled context and kept scanning")
	}
}
