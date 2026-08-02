package engine

import (
	"errors"
	"testing"

	"github.com/tsvsheet/go-isnow/internal/constants"
)

// TestBoundedStridesRejectsAStrideThatCannotProgress pins the refusal. A stride
// at least as large as its cycle collapses to the anchor — "every 60 minutes"
// inside a 60-minute cycle names one minute forever — so it is a range error
// rather than a pattern that silently means something much narrower than it
// reads. Cross-cycle periods have their own spelling (+[90m]).
func TestBoundedStridesRejectsAStrideThatCannotProgress(t *testing.T) {
	t.Parallel()
	if err := boundedStrides(roleMinute, []count{59}); err != nil {
		t.Fatalf("a stride inside its cycle is fine: %v", err)
	}
	if err := boundedStrides(roleMinute, []count{60}); !errors.Is(err, constants.ErrRange) {
		t.Fatalf("stride == cycle: err = %v, want ErrRange", err)
	}
	if err := boundedStrides(roleMinute, []count{120}); !errors.Is(err, constants.ErrRange) {
		t.Fatalf("stride > cycle: err = %v, want ErrRange", err)
	}
	if err := boundedStrides(roleYear, []count{1000}); err != nil {
		t.Fatalf("a year is an open progression with no cycle to exceed: %v", err)
	}
}
