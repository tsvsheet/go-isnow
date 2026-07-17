package engine

import "time"

// PatternText is isnow pattern source text accepted by Parse. It is exported so
// callers in other packages can convert their string input at the call site.
type PatternText string

// Pattern is a parsed, canonicalized isnow. The zero value is not usable; obtain
// one from Parse. Patterns are immutable and safe to copy and share.
type Pattern struct {
	canon       string
	explanation string
	bounds      []boundSpec
	intervals   []intervalSpec
	exclusions  []exclusionSpec
	fields      [numRoles]fieldSpec
}

// Parse recognizes src, resolves symbols and the shorthand ladder, and validates
// field domains. It returns one of constants.ErrSyntax, constants.ErrSymbol, constants.ErrRange, or constants.ErrContext.
func Parse(src PatternText) (Pattern, error) {
	raw, err := parseRaw(src)
	if err != nil {
		return Pattern{}, err
	}
	if verr := validateUnits(raw); verr != nil {
		return Pattern{}, verr
	}
	sl, err := mapGroups(raw.groups, wildDefault(hasSecondInterval(raw.intervals)), false)
	if err != nil {
		return Pattern{}, err
	}
	return assemble(sl, raw)
}

func assemble(sl slots, raw rawPattern) (Pattern, error) {
	fields, err := compileAll(sl)
	if err != nil {
		return Pattern{}, err
	}
	bounds, err := compileBounds(raw.bounds)
	if err != nil {
		return Pattern{}, err
	}
	intervals, err := compileIntervals(raw.intervals)
	if err != nil {
		return Pattern{}, err
	}
	exclusions, err := compileExclusions(raw.exclusions)
	if err != nil {
		return Pattern{}, err
	}
	return Pattern{
		fields:      fields,
		bounds:      bounds,
		intervals:   intervals,
		exclusions:  exclusions,
		canon:       renderCanonical(sl, intervals, bounds) + renderExclusions(exclusions),
		explanation: renderExplain(sl, bounds),
	}, nil
}

func compileIntervals(raw []rawIncr) ([]intervalSpec, error) {
	out := make([]intervalSpec, len(raw))
	for i, in := range raw {
		iv, err := compileInterval(in)
		if err != nil {
			return nil, err
		}
		out[i] = iv
	}
	return out, nil
}

func renderIntervals(ivs []intervalSpec) string {
	s := ""
	for _, iv := range ivs {
		s += " " + iv.text
	}
	return s
}

func compileAll(sl slots) ([numRoles]fieldSpec, error) {
	var out [numRoles]fieldSpec
	for r := role(0); r < numRoles; r++ {
		spec, err := compileRole(r, sl[r])
		if err != nil {
			return out, err
		}
		out[r] = spec
	}
	return out, nil
}

func compileRole(r role, sl slot) (fieldSpec, error) {
	if !sl.isAssigned || !sl.field.isPresent {
		return wildcardField(), nil
	}
	return compileField(r, sl.field)
}

// Canonical returns the fully-qualified seven-field form of the isnow.
func (p Pattern) Canonical() string { return p.canon }

// String returns the canonical form (fmt.Stringer).
func (p Pattern) String() string { return p.canon }

// Is is the one-shot membership test: Parse then Holds.
func Is(src PatternText, at time.Time) (bool, error) {
	p, err := Parse(src)
	if err != nil {
		return false, err
	}
	return p.Holds(at), nil
}
