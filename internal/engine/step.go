package engine

import (
	"slices"

	"github.com/tsvsheet/go-isnow/internal/constants"
)

// anchoredAtEnd marks a '-' step or magnitude: anchored at the cycle end,
// descending.
type anchoredAtEnd bool

// anchorElided marks a step whose anchor is unwritten (or '*'), so it anchors
// at the cycle edge.
type anchorElided bool

// stepTerm dispatches a step to week-granular, weekday-occurrence, or arithmetic
// interpretation per the anchor and unit (specs/contracts/semantics.md §Step).
func stepTerm(r role, anchor rawAtom, in rawIncr) (pred, error) {
	switch {
	case hasWeekUnit(in):
		return weekStepTerm(r, anchor, in)
	case r == roleWeekday && anchor.isPresent && anchor.name != "":
		return occurrenceTerm(anchor.name, in)
	default:
		return arithStepTerm(r, anchor, in)
	}
}

func hasWeekUnit(in rawIncr) bool {
	for _, q := range in.qtys {
		if q.unit == "w" {
			return true
		}
	}
	return false
}

// occurrenceTerm selects the nth <weekday> of the month (or nth from the end).
func occurrenceTerm(name symbolName, in rawIncr) (pred, error) {
	set, res := resolveWeekday(name)
	if res != resOne {
		return nil, constants.ErrSymbol
	}
	ks := qtyNums(in.qtys)
	if err := occurrenceIndices(ks); err != nil {
		return nil, err
	}
	return func(c instantCtx) bool {
		if !slices.Contains(set, c.value(roleWeekday)) {
			return false
		}
		idx := c.occ
		if in.isFromEnd {
			idx = c.occFromEnd
		}
		return slices.Contains(ks, count(idx))
	}, nil
}

// occurrenceIndices validates weekday-occurrence selectors: a month holds at
// most five of any weekday, so the index must be 1..5.
func occurrenceIndices(ks []count) error {
	for _, k := range ks {
		if k < 1 || k > 5 {
			return constants.ErrRange
		}
	}
	return nil
}

// weekStepTerm matches days whose zero-based week-of-year index is ≡ anchor mod N.
func weekStepTerm(r role, anchor rawAtom, in rawIncr) (pred, error) {
	if r != roleDay {
		return nil, constants.ErrContext
	}
	a, err := anchorNum(anchor)
	if err != nil {
		return nil, err
	}
	n, err := stepN(in)
	if err != nil {
		return nil, err
	}
	if count(a) >= n || n > weeksPerYear {
		return nil, constants.ErrRange // a week stride must be 1..53 and larger than its anchor
	}
	return func(c instantCtx) bool {
		wi := count((c.dayOfYear - 1) / 7)
		return mod(wi-count(a), n) == 0
	}, nil
}

// weeksPerYear caps a week-granular step; a year spans at most 53 week buckets.
const weeksPerYear = 53

// arithStepTerm matches an arithmetic progression from the anchor (or the cycle
// edge when the anchor is elided). A '-' step descends from the anchor/cycle end.
func arithStepTerm(r role, anchor rawAtom, in rawIncr) (pred, error) {
	a, isElided, err := anchorOrElided(r, anchor, in.isFromEnd)
	if err != nil {
		return nil, err
	}
	ns, err := stepNs(in)
	if err != nil {
		return nil, err
	}
	if err := boundedStrides(r, ns); err != nil {
		return nil, err
	}
	return func(c instantCtx) bool { return anyStep(c, r, a, isElided, in.isFromEnd, ns) }, nil
}

// boundedStrides rejects a field-local stride that cannot progress within its
// cycle (stride >= cycle collapses to the anchor). Cross-cycle periods use a
// unit step (e.g. +[90m]) instead. Year steps are open progressions (no cycle).
func boundedStrides(r role, ns []count) error {
	cs := cycleSize(r)
	if cs == 0 {
		return nil
	}
	for _, n := range ns {
		if n >= cs {
			return constants.ErrRange
		}
	}
	return nil
}

func anyStep(c instantCtx, r role, anchor fieldValue, isElided anchorElided, isFromEnd anchoredAtEnd, ns []count) bool {
	base := anchor
	if isElided {
		clo, chi := c.cycle(r)
		base = edge(clo, chi, isFromEnd)
	}
	v := c.value(r)
	for _, n := range ns {
		if stepHit(v, base, isFromEnd, n) {
			return true
		}
	}
	return false
}

func edge(clo, chi fieldValue, isFromEnd anchoredAtEnd) fieldValue {
	if isFromEnd {
		return chi
	}
	return clo
}

func stepHit(v, base fieldValue, isFromEnd anchoredAtEnd, n count) bool {
	if isFromEnd {
		return v <= base && count(base-v)%n == 0
	}
	return v >= base && count(v-base)%n == 0
}

// anchorOrElided resolves a numeric step anchor. Year steps that would need a
// cycle (an elided or from-end anchor) are rejected: year has no natural cycle
// and the window-as-cycle feature is deferred (decision 004).
func anchorOrElided(r role, anchor rawAtom, isFromEnd anchoredAtEnd) (fieldValue, anchorElided, error) {
	if !anchor.isPresent || anchor.isStar {
		return 0, true, yearGuard(r)
	}
	a, err := anchorNum(anchor)
	if err != nil {
		return 0, false, err
	}
	if isFromEnd {
		return a, false, yearGuard(r)
	}
	return a, false, nil
}

func yearGuard(r role) error {
	if r == roleYear {
		return constants.ErrContext
	}
	return nil
}

func anchorNum(a rawAtom) (fieldValue, error) {
	if !a.isPresent || a.isStar {
		return 0, nil
	}
	if a.name != "" || len(a.qtys) != 1 || a.qtys[0].unit != "" {
		return 0, constants.ErrContext // a step anchor is a plain number, not a unit compound
	}
	v := fieldValue(a.qtys[0].num)
	if err := anchorDomain(v); err != nil {
		return 0, err
	}
	return v, nil
}

// anchorDomain rejects absurd anchors (negatives can't occur; huge values are range).
func anchorDomain(v fieldValue) error {
	if v < 0 || v > 9999 {
		return constants.ErrRange
	}
	return nil
}

func stepN(in rawIncr) (count, error) {
	return positive(count(in.qtys[0].num))
}

func stepNs(in rawIncr) ([]count, error) {
	out := make([]count, len(in.qtys))
	for i, q := range in.qtys {
		n, err := positive(count(q.num))
		if err != nil {
			return nil, err
		}
		out[i] = n
	}
	return out, nil
}

func positive(n count) (count, error) {
	if n < 1 {
		return 0, constants.ErrRange
	}
	return n, nil
}

func qtyNums(qs []rawQty) []count {
	out := make([]count, len(qs))
	for i, q := range qs {
		out[i] = count(q.num)
	}
	return out
}

func mod(a, n count) count {
	return ((a % n) + n) % n
}

// spanStepTerm restricts an arithmetic step to an inclusive span.
func spanStepTerm(r role, t rawTerm) (pred, error) {
	sp, err := spanTerm(r, t)
	if err != nil {
		return nil, err
	}
	st, err := arithStepTerm(r, t.lo, t.incr)
	if err != nil {
		return nil, err
	}
	return func(c instantCtx) bool { return sp(c) && st(c) }, nil
}
