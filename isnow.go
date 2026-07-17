package isnow

import (
	"time"

	"github.com/tsvsheet/go-isnow/internal/engine"
)

// PatternText is isnow pattern source text accepted by Parse. It is exported so
// callers in other packages can convert their string input at the call site.
type PatternText = engine.PatternText

// Pattern is a parsed, canonicalized isnow. The zero value is not usable; obtain
// one from Parse. Patterns are immutable and safe to copy and share.
type Pattern = engine.Pattern

// Parse recognizes src, resolves symbols and the shorthand ladder, and validates
// field domains. It returns one of ErrSyntax, ErrSymbol, ErrRange, or ErrContext.
func Parse(src PatternText) (Pattern, error) { return engine.Parse(src) }

// Is is the one-shot membership test: Parse then Holds.
func Is(src PatternText, at time.Time) (bool, error) { return engine.Is(src, at) }

// Code returns the stable string code of a library error (syntax, symbol,
// range, context), or "" if err is nil or not a library error.
func Code(err error) string { return engine.Code(err) }
