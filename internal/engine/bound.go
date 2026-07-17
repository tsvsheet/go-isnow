package engine

import "github.com/tsvsheet/go-isnow/internal/constants"

// boundField is one constrained (exact) field of a bound sub-spec.
type boundField struct {
	role  role
	value fieldValue
}

// boundSpec is a compiled since/until bound: positional tuple comparison over
// its constrained fields (specs/contracts/semantics.md §Bounds). text is the
// canonical sub-spec, rendered as a full Y/m/d w H:M:S form so it round-trips.
type boundSpec struct {
	text   string
	fields []boundField
	op     boundKind
}

func (b boundSpec) satisfied(c instantCtx) bool {
	cmp := b.compare(c)
	switch b.op {
	case geBound:
		return cmp >= 0
	case gtBound:
		return cmp > 0
	case leBound:
		return cmp <= 0
	default:
		return cmp < 0
	}
}

func (b boundSpec) compare(c instantCtx) int {
	for _, bf := range b.fields {
		v := c.value(bf.role)
		if v < bf.value {
			return -1
		}
		if v > bf.value {
			return 1
		}
	}
	return 0
}

func compileBounds(raw []rawBound) ([]boundSpec, error) {
	out := make([]boundSpec, len(raw))
	for i := range raw {
		b, err := compileBound(raw[i])
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

func compileBound(b rawBound) (boundSpec, error) {
	sl, err := mapBoundGroups(b.groups)
	if err != nil {
		return boundSpec{}, err
	}
	fields, err := boundFields(sl)
	if err != nil {
		return boundSpec{}, err
	}
	if len(fields) == 0 {
		return boundSpec{}, constants.ErrContext // a bound must constrain something
	}
	return boundSpec{op: b.op, fields: fields, text: renderCanonical(sl, nil, nil)}, nil
}

// mapBoundGroups maps a bound sub-spec's groups: date/time as usual, a bare
// number to year (four digits) or hour, everything else a context error.
func mapBoundGroups(groups []rawGroup) (slots, error) {
	var s slots
	for _, gr := range groups {
		next, err := assignBoundGroup(s, gr)
		if err != nil {
			return s, err
		}
		s = next
	}
	if err := validateBoundTime(s); err != nil {
		return s, err
	}
	return fillBoundDefaults(s), nil // zero-fill finer time fields so >=6 compares as (6,0,0)
}

// validateBoundTime requires a time bound to be anchored at the hour (and a
// second at the minute): a finer-only time bound would render ambiguously.
func validateBoundTime(s slots) error {
	if !boundHas(s, roleHour) && (boundHas(s, roleMinute) || boundHas(s, roleSecond)) {
		return constants.ErrContext
	}
	if !boundHas(s, roleMinute) && boundHas(s, roleSecond) {
		return constants.ErrContext
	}
	return nil
}

// fillBoundDefaults zero-fills the time fields finer than a provided (present
// and non-empty) one, so a time bound compares as a whole-of-day instant, while
// a year/date bound compares only at its own granularity.
func fillBoundDefaults(s slots) slots {
	if boundHas(s, roleHour) && !s[roleMinute].isAssigned {
		s[roleMinute] = slot{isAssigned: true, field: exactRaw(0)}
	}
	if (boundHas(s, roleHour) || boundHas(s, roleMinute)) && !s[roleSecond].isAssigned {
		s[roleSecond] = slot{isAssigned: true, field: exactRaw(0)}
	}
	return s
}

func boundHas(s slots, r role) bool {
	return s[r].isAssigned && s[r].field.isPresent
}

func assignBoundGroup(s slots, gr rawGroup) (slots, error) {
	switch gr.kind {
	case dateKind:
		return assignDate(s, gr)
	case timeKind:
		return assignTime(s, gr)
	default:
		return assignBoundBare(s, gr.slots[0])
	}
}

func assignBoundBare(s slots, f rawField) (slots, error) {
	a := firstAtom(f)
	if a.isStar {
		return s, nil // a bare wildcard is no constraint
	}
	if !a.isPresent || a.name != "" || len(a.qtys) == 0 {
		return s, constants.ErrContext
	}
	if a.qtys[0].digits == 4 {
		return claim(s, roleYear, f)
	}
	return claim(s, roleHour, f)
}

// boundFields extracts the constrained exact values in role order, rejecting
// weekday constraints and any non-exact algebra.
func boundFields(s slots) ([]boundField, error) {
	var out []boundField
	for r := role(0); r < numRoles; r++ {
		bf, ok, err := boundValue(r, s[r])
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, bf)
		}
	}
	return out, nil
}

func boundValue(r role, sl slot) (boundField, bool, error) {
	if !sl.isAssigned || !sl.field.isPresent || isWildcardField(sl.field) {
		return boundField{}, false, nil
	}
	if r == roleWeekday || !isExactField(sl.field) {
		return boundField{}, false, constants.ErrContext
	}
	v := fieldValue(sl.field.terms[0].lo.qtys[0].num)
	if err := inDomain(r, v); err != nil {
		return boundField{}, false, err
	}
	return boundField{role: r, value: v}, true, nil
}

// isWildcardField reports whether a field is a lone wildcard (no constraint).
func isWildcardField(f rawField) bool {
	if f.isExcluded || len(f.terms) != 1 {
		return false
	}
	t := f.terms[0]
	return t.lo.isStar && !t.hi.isPresent && !t.incr.isPresent && !bool(t.isLoFromEnd)
}

// isExactField reports whether a field is a single plain numeric value.
func isExactField(f rawField) bool {
	if f.isExcluded || len(f.terms) != 1 {
		return false
	}
	t := f.terms[0]
	return t.lo.isPresent && !t.lo.isStar && t.lo.name == "" &&
		!t.hi.isPresent && !t.incr.isPresent && !bool(t.isLoFromEnd) &&
		len(t.lo.qtys) == 1
}
