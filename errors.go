package isnow

import "github.com/tsvsheet/go-isnow/internal/constants"

// The four stable error codes every implementation shares (specs/contracts/
// semantics.md in the grammar repo). Callers match with errors.Is.
const (
	// ErrSyntax is a malformed pattern the grammar rejects.
	ErrSyntax = constants.ErrSyntax
	// ErrSymbol is an unknown or ambiguous weekday/time/unit name.
	ErrSymbol = constants.ErrSymbol
	// ErrRange is a value outside its field's domain.
	ErrRange = constants.ErrRange
	// ErrContext is a grammatical but semantically invalid construct.
	ErrContext = constants.ErrContext
)
