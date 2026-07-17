// Package constants declares the isnow library's sentinel error values — the
// four stable error codes every isnow implementation shares
// (specs/contracts/semantics.md in the grammar repo). The error mechanism (the
// matchable string type) lives in the shared gomatic/go-error library; these
// values are this package's own. Callers match with errors.Is.
package constants

import errs "github.com/gomatic/go-error"

// Keep these constants sorted alphabetically.
const (
	// ErrContext is a grammatical but semantically invalid construct.
	ErrContext errs.Const = "context"
	// ErrRange is a value outside its field's domain.
	ErrRange errs.Const = "range"
	// ErrSymbol is an unknown or ambiguous weekday/time/unit name.
	ErrSymbol errs.Const = "symbol"
	// ErrSyntax is a malformed pattern the grammar rejects.
	ErrSyntax errs.Const = "syntax"
)
