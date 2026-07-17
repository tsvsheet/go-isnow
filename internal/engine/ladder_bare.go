package engine

import "github.com/tsvsheet/go-isnow/internal/constants"

type bareRoute int

const (
	routeWeekday bareRoute = iota
	routeTime
	routeNumber
)

// assignBare routes a bare group's field to a role by its content.
func assignBare(s slots, f rawField, isThreeForm threeGroupForm) (slots, error) {
	route, err := classifyBare(f)
	if err != nil {
		return s, err
	}
	switch route {
	case routeWeekday:
		return claim(s, roleWeekday, f)
	case routeTime:
		return assignTimeSymbol(s, f)
	default:
		return assignBareNumber(s, f, isThreeForm)
	}
}

// classifyBare inspects the first atom: a weekday symbol or '*' routes to
// weekday, a time symbol to the time group, a number to hour/weekday.
func classifyBare(f rawField) (bareRoute, error) {
	a := firstAtom(f)
	switch {
	case !a.isPresent || a.isStar:
		return routeWeekday, nil
	case a.name != "":
		return classifyName(a.name)
	default:
		return routeNumber, nil
	}
}

func classifyName(name symbolName) (bareRoute, error) {
	if _, res := resolveWeekday(name); res == resOne {
		return routeWeekday, nil
	} else if res == resAmbiguous {
		return routeWeekday, constants.ErrSymbol
	}
	switch _, res := resolveTime(name); res {
	case resOne:
		return routeTime, nil
	default:
		return routeTime, constants.ErrSymbol
	}
}

// firstAtom returns the first term's lo atom (absent for an incr-only term);
// the grammar guarantees a present field has at least one term.
func firstAtom(f rawField) rawAtom {
	return f.terms[0].lo
}

// assignTimeSymbol expands a bare time symbol into exact H:M:S fields. The
// caller (classifyBare) has already confirmed the symbol resolves uniquely.
func assignTimeSymbol(s slots, f rawField) (slots, error) {
	hms, _ := resolveTime(f.terms[0].lo.name)
	for i, r := range []role{roleHour, roleMinute, roleSecond} {
		next, err := claim(s, r, exactRaw(fieldValue(hms[i])))
		if err != nil {
			return s, err
		}
		s = next
	}
	return s, nil
}

// assignBareNumber fills an unconstrained hour slot (unassigned or
// assigned-but-empty). If the hour is explicitly constrained, the number is a
// weekday only in the full three-group form; otherwise it is a context error.
func assignBareNumber(s slots, f rawField, isThreeForm threeGroupForm) (slots, error) {
	if hourFree(s) {
		s[roleHour] = slot{isAssigned: true, field: f}
		return s, nil
	}
	if isThreeForm {
		return claim(s, roleWeekday, f)
	}
	return s, constants.ErrContext
}

func hourFree(s slots) bool {
	return !s[roleHour].isAssigned || !s[roleHour].field.isPresent
}

// fillDefaults assigns wildcards and time defaults to unassigned roles.
// isSecondWild makes an unprovided second default to wildcard (a second-grained
// interval owns the second field) rather than to 0.
func fillDefaults(s slots, isSecondWild, isTimeWild wildDefault) slots {
	for _, r := range []role{roleYear, roleMonth, roleDay, roleWeekday} {
		if !s[r].isAssigned {
			s[r] = slot{isAssigned: true}
		}
	}
	return fillTimeDefaults(s, isSecondWild, isTimeWild)
}

// fillTimeDefaults applies the time-default rule: when no time field is provided
// at all, the time is *:*:00; otherwise unassigned time roles (always finer than
// the coarsest provided one) default to 0.
func fillTimeDefaults(s slots, isSecondWild, isTimeWild wildDefault) slots {
	roles := []role{roleHour, roleMinute, roleSecond}
	if !anyProvided(s, roles) {
		s[roleHour], s[roleMinute] = slot{isAssigned: true}, slot{isAssigned: true}
		s[roleSecond] = defaultSecond(isSecondWild || isTimeWild)
		return s
	}
	for _, r := range roles {
		if !s[r].isAssigned {
			s[r] = slot{isAssigned: true, field: exactRaw(0)}
		}
	}
	return s
}

func defaultSecond(isWild wildDefault) slot {
	if isWild {
		return slot{isAssigned: true}
	}
	return slot{isAssigned: true, field: exactRaw(0)}
}

func anyProvided(s slots, roles []role) bool {
	for _, r := range roles {
		if s[r].isAssigned {
			return true
		}
	}
	return false
}
