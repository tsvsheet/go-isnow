package engine

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/tsvsheet/go-isnow/internal/constants"
	g "github.com/tsvsheet/go-isnow/internal/isnowgrammar"
)

// groupKind distinguishes the three group shapes the grammar produces.
type groupKind int

const (
	dateKind groupKind = iota
	timeKind
	bareKind
)

// boundKind is a since/until comparator.
type boundKind int

const (
	geBound boundKind = iota
	gtBound
	leBound
	ltBound
)

// rawQty is a magnitude with an optional unit name ('w'/'d'); digits is the
// written digit count, which distinguishes a four-digit year in a bound.
type rawQty struct {
	unit   unitName
	num    int
	digits int
}

// rawAtom is a single value: wildcard, a NAME symbol, or a numeric quantity
// run. The zero value is an absent atom; the parser marks every parsed atom
// present.
type rawAtom struct {
	name      symbolName
	qtys      []rawQty
	isStar    bool
	isPresent bool
}

// rawIncr is a step expression, '+' (from anchor) or '-' (from the end). The
// zero value is an absent step; the parser marks every parsed step present.
type rawIncr struct {
	qtys      []rawQty
	isFromEnd anchoredAtEnd
	isPresent bool
}

// rawTerm is the shared per-field algebra `!v-v±[N]` (exclusion held on the field).
type rawTerm struct {
	lo          rawAtom
	hi          rawAtom
	incr        rawIncr
	isLoFromEnd anchoredAtEnd
}

// rawField is an optional exclusion over a set of terms; absent = empty slot.
type rawField struct {
	terms      []rawTerm
	isPresent  bool
	isExcluded bool
}

// rawGroup is one date/time/bare group with its positional field slots.
type rawGroup struct {
	slots []rawField
	kind  groupKind
}

// rawBound is one since/until bound with its sub-spec groups.
type rawBound struct {
	groups []rawGroup
	op     boundKind
}

// rawPattern is the whole parse: the main groups, any bounds, intervals, and
// pattern-level exclusions (each a sub-spec of groups).
type rawPattern struct {
	groups     []rawGroup
	bounds     []rawBound
	intervals  []rawIncr
	exclusions [][]rawGroup
}

// tokenIndex is a token's position in the lexer's token stream, used to place
// positional fields relative to their separators.
type tokenIndex int

// errListener records the first syntax error so parsing yields constants.ErrSyntax rather
// than antlr's default stderr print. The flag is held through a pointer so the
// antlr callback, which may receive the listener by value, still lands the
// write while the type keeps value receivers.
type errListener struct {
	*antlr.DefaultErrorListener
	hasFailed *bool
}

func (l errListener) SyntaxError(antlr.Recognizer, any, int, int, string, antlr.RecognitionException) {
	*l.hasFailed = true
}

// parseTree runs the generated lexer/parser with error listeners replaced.
func parseTree(src PatternText) (g.IPatternContext, bool) {
	failed := false
	listener := errListener{DefaultErrorListener: antlr.NewDefaultErrorListener(), hasFailed: &failed}
	input := antlr.NewInputStream(string(src))
	lexer := g.NewIsnowLexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(listener)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := g.NewIsnowParser(stream)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(listener)
	tree := parser.Pattern()
	return tree, !failed
}

// parseRaw parses src into the raw AST, or constants.ErrSyntax.
func parseRaw(src PatternText) (rawPattern, error) {
	tree, ok := parseTree(src)
	if !ok {
		return rawPattern{}, constants.ErrSyntax
	}
	groups, intervals := extractIntervals(specGroups(tree.Spec()))
	return rawPattern{
		groups:     groups,
		bounds:     bounds(tree.AllBound()),
		intervals:  intervals,
		exclusions: exclusions(tree.AllExclusion()),
	}, nil
}

func exclusions(ctxs []g.IExclusionContext) [][]rawGroup {
	out := make([][]rawGroup, len(ctxs))
	for i, ctx := range ctxs {
		out[i] = specGroups(ctx.Spec())
	}
	return out
}

// extractIntervals pulls out the bare interval groups (`+[90mn]`, `+[10d]`, …)
// so they become pattern-level periodic constraints rather than field terms.
func extractIntervals(groups []rawGroup) ([]rawGroup, []rawIncr) {
	var kept []rawGroup
	var intervals []rawIncr
	for _, gr := range groups {
		if in, ok := intervalOf(gr); ok {
			intervals = append(intervals, in)
			continue
		}
		kept = append(kept, gr)
	}
	return kept, intervals
}

// intervalOf returns the interval increment if gr is a bare group holding a
// single incr-only term with an interval unit (s/mn/h/d).
func intervalOf(gr rawGroup) (rawIncr, bool) {
	if gr.kind != bareKind {
		return rawIncr{}, false
	}
	f := gr.slots[0]
	if !f.isPresent || f.isExcluded || len(f.terms) != 1 {
		return rawIncr{}, false
	}
	t := f.terms[0]
	if t.lo.isPresent || !t.incr.isPresent || len(t.incr.qtys) != 1 {
		return rawIncr{}, false
	}
	if _, ok := intervalUnits[t.incr.qtys[0].unit]; !ok {
		return rawIncr{}, false
	}
	return t.incr, true
}

func bounds(ctxs []g.IBoundContext) []rawBound {
	out := make([]rawBound, len(ctxs))
	for i, ctx := range ctxs {
		out[i] = rawBound{op: boundOp(ctx.BoundOp()), groups: specGroups(ctx.Spec())}
	}
	return out
}

func boundOp(ctx g.IBoundOpContext) boundKind {
	switch {
	case ctx.GE() != nil:
		return geBound
	case ctx.GT() != nil:
		return gtBound
	case ctx.LE() != nil:
		return leBound
	default:
		return ltBound
	}
}

func specGroups(spec g.ISpecContext) []rawGroup {
	ctxs := spec.AllGroup()
	out := make([]rawGroup, len(ctxs))
	for i, ctx := range ctxs {
		out[i] = group(ctx)
	}
	return out
}

func group(ctx g.IGroupContext) rawGroup {
	switch node := ctx.GetChild(0).(type) {
	case g.IDateGroupContext:
		return rawGroup{kind: dateKind, slots: dateSlots(node)}
	case g.ITimeGroupContext:
		return rawGroup{kind: timeKind, slots: timeSlots(node)}
	default:
		return rawGroup{kind: bareKind, slots: []rawField{field(node.(g.IBareGroupContext).Field())}}
	}
}

// dateSlots and timeSlots read the present-or-empty field slots. The grammar's
// `field? (SEP field?)+` means slot count = separators + 1; an omitted field is
// an empty (positionally-wildcard) slot. A present field's slot index is the
// number of separators appearing before it in the token stream.
func dateSlots(ctx g.IDateGroupContext) []rawField {
	return placeFields(ctx.AllField(), sepIndices(ctx.AllSLASH()))
}

func timeSlots(ctx g.ITimeGroupContext) []rawField {
	return placeFields(ctx.AllField(), sepIndices(ctx.AllCOLON()))
}

func sepIndices(seps []antlr.TerminalNode) []tokenIndex {
	out := make([]tokenIndex, len(seps))
	for i, s := range seps {
		out[i] = tokenIndex(s.GetSymbol().GetTokenIndex())
	}
	return out
}

// placeFields assigns each present field to the slot after the separators that
// precede it, leaving the rest empty. slotCount = len(seps)+1.
func placeFields(fields []g.IFieldContext, seps []tokenIndex) []rawField {
	out := make([]rawField, len(seps)+1)
	for _, fc := range fields {
		out[slotOf(tokenIndex(fc.GetStart().GetTokenIndex()), seps)] = field(fc)
	}
	return out
}

func slotOf(fieldTok tokenIndex, seps []tokenIndex) int {
	idx := 0
	for _, s := range seps {
		if s < fieldTok {
			idx++
		}
	}
	return idx
}
