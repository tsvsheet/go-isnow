package engine

import "github.com/tsvsheet/go-isnow/internal/constants"

// slot is one role's assignment state: unassigned, assigned-empty (an omitted
// positional field, matching anything), or assigned to a present raw field.
type slot struct {
	field      rawField
	isAssigned bool
}

// slots holds the slot assigned to each role.
type slots [numRoles]slot

// threeGroupForm marks the full date+time form, in which a bare number may
// claim the weekday once the hour is taken.
type threeGroupForm bool

// barePass selects which of the two assignment passes runs: date/time groups
// first, then bare groups.
type barePass bool

// wildDefault makes an unprovided time field default to wildcard rather than 0.
type wildDefault bool

// exactRaw builds a present field of a single exact value (for defaults and time
// symbols).
func exactRaw(v fieldValue) rawField {
	return rawField{
		isPresent: true,
		terms:     []rawTerm{{lo: rawAtom{isPresent: true, qtys: []rawQty{{num: int(v), digits: 1}}}}},
	}
}

// mapGroups applies the shorthand ladder: it assigns each group's fields to the
// seven roles, then fills defaults (specs/contracts/semantics.md §Ladder).
func mapGroups(groups []rawGroup, isSecondWild, isTimeWild wildDefault) (slots, error) {
	hasDate, hasTime := kinds(groups)
	s, err := assignGroups(slots{}, groups, threeGroupForm(hasDate && hasTime))
	if err != nil {
		return s, err
	}
	return fillDefaults(s, isSecondWild, isTimeWild), nil
}

func kinds(groups []rawGroup) (hasDate, hasTime bool) {
	for _, gr := range groups {
		switch gr.kind {
		case dateKind:
			hasDate = true
		case timeKind:
			hasTime = true
		}
	}
	return hasDate, hasTime
}

// assignGroups assigns date and time groups before bare groups, so a bare
// number sees whether the hour is already explicitly constrained.
func assignGroups(s slots, groups []rawGroup, isThreeForm threeGroupForm) (slots, error) {
	s, err := assignPass(s, groups, false, isThreeForm)
	if err != nil {
		return s, err
	}
	return assignPass(s, groups, true, isThreeForm)
}

func assignPass(s slots, groups []rawGroup, isBareOnly barePass, isThreeForm threeGroupForm) (slots, error) {
	for _, gr := range groups {
		if (gr.kind == bareKind) != bool(isBareOnly) {
			continue
		}
		next, err := assignGroup(s, gr, isThreeForm)
		if err != nil {
			return s, err
		}
		s = next
	}
	return s, nil
}

func assignGroup(s slots, gr rawGroup, isThreeForm threeGroupForm) (slots, error) {
	switch gr.kind {
	case dateKind:
		return assignDate(s, gr)
	case timeKind:
		return assignTime(s, gr)
	default:
		return assignBare(s, gr.slots[0], isThreeForm)
	}
}

func assignDate(s slots, gr rawGroup) (slots, error) {
	roles, err := dateRoles(count(len(gr.slots)))
	if err != nil {
		return s, err
	}
	return assignPositional(s, roles, gr.slots)
}

func assignTime(s slots, gr rawGroup) (slots, error) {
	roles, err := timeRoles(count(len(gr.slots)))
	if err != nil {
		return s, err
	}
	return assignPositional(s, roles, gr.slots)
}

func dateRoles(n count) ([]role, error) {
	switch n {
	case 2:
		return []role{roleMonth, roleDay}, nil
	case 3:
		return []role{roleYear, roleMonth, roleDay}, nil
	default:
		return nil, constants.ErrContext
	}
}

func timeRoles(n count) ([]role, error) {
	switch n {
	case 2:
		return []role{roleHour, roleMinute}, nil
	case 3:
		return []role{roleHour, roleMinute, roleSecond}, nil
	default:
		return nil, constants.ErrContext
	}
}

func assignPositional(s slots, roles []role, fields []rawField) (slots, error) {
	for i, r := range roles {
		next, err := claim(s, r, fields[i])
		if err != nil {
			return s, err
		}
		s = next
	}
	return s, nil
}

// claim assigns a role, treating a present-but-empty field as a wildcard and
// rejecting a double assignment.
func claim(s slots, r role, f rawField) (slots, error) {
	if s[r].isAssigned {
		return s, constants.ErrContext
	}
	s[r] = slot{isAssigned: true, field: f}
	return s, nil
}
