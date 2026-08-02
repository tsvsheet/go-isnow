package engine

import "testing"

// TestSymbolNameMatchesByUnambiguousPrefixOnly pins the abbreviation rule. A
// prefix that could mean two symbols must be refused rather than resolved to
// whichever happens to be first in the table: "s" is Saturday or Sunday, and a
// pattern that silently picked one would fire on the wrong day every week.
func TestSymbolNameMatchesByUnambiguousPrefixOnly(t *testing.T) {
	t.Parallel()
	for _, src := range []PatternText{"*/*/* tu 12:00", "*/*/* TUE 12:00", "*/*/* tuesday 12:00"} {
		if _, err := Parse(src); err != nil {
			t.Errorf("Parse(%q): an unambiguous prefix resolves: %v", src, err)
		}
	}

	for _, src := range []PatternText{"*/*/* s 12:00", "*/*/* t 12:00"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q): an ambiguous prefix must be refused, not guessed", src)
		}
	}
}
