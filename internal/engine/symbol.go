package engine

import "strings"

// symbolName is a NAME token's text: a weekday or time-of-day symbol, matched
// case-insensitively and by unambiguous prefix.
type symbolName string

// weekdayName is a full lowercase weekday word.
type weekdayName string

// Weekday numbering is Sunday = 1 .. Saturday = 7 (specs/contracts/semantics.md).
var weekdayNames = []weekdayName{"", "sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

// weekdayRuns are the fixed multi-day tokens that are not name prefixes.
var weekdayRuns = map[symbolName][]fieldValue{
	"mwf": {2, 4, 6},
	"ss":  {1, 7},
	"tt":  {3, 5},
}

// timeSymbols maps each time word to its H:M:S; abbreviations resolve directly.
var timeSymbols = map[symbolName][3]int{
	"midnight": {0, 0, 0},
	"noon":     {12, 0, 0},
	"midday":   {12, 0, 0},
}
var timeAbbrev = map[symbolName][3]int{"mn": {0, 0, 0}, "md": {12, 0, 0}}

type resolution int

const (
	resNone resolution = iota
	resOne
	resAmbiguous
)

// resolveWeekday returns the weekday set a name denotes. Weekday resolution runs
// before time resolution, so m = Monday wins over midnight.
func resolveWeekday(name symbolName) ([]fieldValue, resolution) {
	low := symbolName(strings.ToLower(string(name)))
	if run, ok := weekdayRuns[low]; ok {
		return run, resOne
	}
	return prefixWeekday(low)
}

func prefixWeekday(low symbolName) ([]fieldValue, resolution) {
	var found []fieldValue
	for n := 1; n <= 7; n++ {
		if strings.HasPrefix(string(weekdayNames[n]), string(low)) {
			found = append(found, fieldValue(n))
		}
	}
	switch len(found) {
	case 0:
		return nil, resNone
	case 1:
		return found, resOne
	default:
		return nil, resAmbiguous
	}
}

// resolveTime returns the H:M:S a time symbol denotes.
func resolveTime(name symbolName) ([3]int, resolution) {
	low := symbolName(strings.ToLower(string(name)))
	if hms, ok := timeAbbrev[low]; ok {
		return hms, resOne
	}
	return prefixTime(low)
}

func prefixTime(low symbolName) ([3]int, resolution) {
	var hits [][3]int
	for word, hms := range timeSymbols {
		if strings.HasPrefix(string(word), string(low)) {
			hits = append(hits, hms)
		}
	}
	return uniqueTime(hits)
}

func uniqueTime(hits [][3]int) ([3]int, resolution) {
	if len(hits) == 0 {
		return [3]int{}, resNone
	}
	for _, h := range hits[1:] {
		if h != hits[0] {
			return [3]int{}, resAmbiguous
		}
	}
	return hits[0], resOne
}
