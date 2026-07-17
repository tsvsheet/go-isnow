package engine

import (
	"errors"

	errs "github.com/gomatic/go-error"

	"github.com/tsvsheet/go-isnow/internal/constants"
)

// Code returns the stable string code of a library error (syntax, symbol,
// range, context), or "" if err is nil or not a library error.
func Code(err error) string {
	for _, c := range []errs.Const{constants.ErrSyntax, constants.ErrSymbol, constants.ErrRange, constants.ErrContext} {
		if errors.Is(err, c) {
			return string(c)
		}
	}
	return ""
}
