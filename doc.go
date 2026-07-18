// Package isnow implements the isnow date/time pattern language: an isnow
// holds at an instant when every field constraint is satisfied. The defining
// operation is the membership test, Pattern.Holds(at); Next/Prev derive
// occurrences from it. See github.com/tsvsheet/isnow (SPECIFICATION.md) for the
// language and specs/contracts/ for the pinned semantics.
//
// Parse recognizes pattern source (PatternText), resolves symbols and the
// shorthand ladder, and validates field domains, returning an immutable
// Pattern that is safe to copy and share across goroutines. Is is the one-shot
// membership test (Parse then Holds). Errors returned to callers are the four
// stable errs.Const sentinels shared by every isnow implementation (ErrSyntax,
// ErrSymbol, ErrRange, ErrContext), matchable with errors.Is; Code maps an
// error back to its stable cross-implementation string code.
//
// The package is filesystem-, network-, and clock-free by construction: every
// query takes the instant to evaluate against as an argument, so callers
// inject their own time source. Pattern recognition reuses the grammar repo's
// ANTLR-generated parser through the internal/isnowgrammar seam; no ANTLR
// type escapes into the public surface.
//
// This package is a thin facade: every type, function, and constant it exposes
// re-exports the implementation in internal/engine unchanged, so the public
// surface is documented here while the engine stays an internal package.
package isnow
