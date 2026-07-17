package engine

import "github.com/tsvsheet/go-isnow/internal/constants"

// unitName is a quantity's unit suffix: 'w'/'d' on a field magnitude, or an
// interval grain ('s'/'mn'/'h'/'d'); empty for a plain number.
type unitName string

// validateUnits rejects any quantity carrying a unit other than 'w' or 'd'
// (specs/contracts/semantics.md: an unknown unit is a symbol error).
func validateUnits(raw rawPattern) error {
	if err := groupsUnits(raw.groups); err != nil {
		return err
	}
	for _, b := range raw.bounds {
		if err := groupsUnits(b.groups); err != nil {
			return err
		}
	}
	for _, ex := range raw.exclusions {
		if err := groupsUnits(ex); err != nil {
			return err
		}
	}
	return nil
}

func groupsUnits(groups []rawGroup) error {
	for _, gr := range groups {
		for _, f := range gr.slots {
			if err := fieldUnits(f); err != nil {
				return err
			}
		}
	}
	return nil
}

func fieldUnits(f rawField) error {
	for _, t := range f.terms {
		if err := termUnits(t); err != nil {
			return err
		}
	}
	return nil
}

func termUnits(t rawTerm) error {
	if err := qtysUnits(t.lo.qtys); err != nil {
		return err
	}
	if err := qtysUnits(t.hi.qtys); err != nil {
		return err
	}
	return qtysUnits(t.incr.qtys)
}

func qtysUnits(qs []rawQty) error {
	for _, q := range qs {
		if q.unit != "" && q.unit != "w" && q.unit != "d" {
			return constants.ErrSymbol
		}
	}
	return nil
}
