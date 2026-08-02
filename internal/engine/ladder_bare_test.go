package engine

import (
	"strings"
	"testing"
)

// TestFirstAtomRestsOnTheGrammarGuaranteeingATerm pins the assumption behind
// an unchecked index. firstAtom reads terms[0] without looking, because a
// present field always has at least one term — the grammar cannot produce an
// empty one. If that ever stopped being true the failure would be an
// out-of-range panic deep in parsing, so the guarantee is asserted here against
// every field a real pattern produces.
func TestFirstAtomRestsOnTheGrammarGuaranteeingATerm(t *testing.T) {
	t.Parallel()
	for _, src := range []PatternText{
		"*/*/* 12:00", "2026/2/3 04:05:06", "M-F noon", "*/*/1,15 09:30",
	} {
		p, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if p.String() == "" {
			t.Fatalf("Parse(%q) produced an unusable pattern", src)
		}
	}
}

// TestFillTimeDefaultsAppliesTheWholeTimeRuleNotHalfOfIt pins both branches of
// the default. A pattern naming no time at all means "any minute of any hour,
// on the minute" — *:*:00 — while one that names an hour and stops means that
// hour exactly, 00 past. Collapsing the two would make `12` mean noon-ish
// rather than noon.
func TestFillTimeDefaultsAppliesTheWholeTimeRuleNotHalfOfIt(t *testing.T) {
	t.Parallel()
	noTime, err := Parse("*/*/*")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	hourOnly, err := Parse("*/*/* 12")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := noTime.String(); !strings.HasSuffix(got, "*:*:00") {
		t.Errorf("a pattern naming no time is *:*:00, got %s", got)
	}
	if got := hourOnly.String(); !strings.HasSuffix(got, "12:00:00") {
		t.Errorf("a pattern naming only an hour fills the rest with zero, got %s", got)
	}
}
