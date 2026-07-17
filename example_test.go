package isnow_test

import (
	"errors"
	"fmt"
	"time"

	isnow "github.com/tsvsheet/go-isnow"
)

// Parse compiles a pattern to an immutable Pattern; Holds is the membership
// test against any instant the caller supplies.
func ExampleParse() {
	p, err := isnow.Parse("M,W,F noon")
	if err != nil {
		fmt.Println(err)
		return
	}
	monday := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	fmt.Println(p.Holds(monday))
	fmt.Println(p.Canonical())
	// Output:
	// true
	// */*/* Monday,Wednesday,Friday 12:00:00
}

// Is is the one-shot form: Parse then Holds.
func ExampleIs() {
	at := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	ok, _ := isnow.Is("F mn", at)
	fmt.Println(ok)
	// Output: true
}

// Next derives the earliest occurrence strictly after an instant.
func ExamplePattern_Next() {
	p, _ := isnow.Parse("12/25 mn")
	from := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	next, ok := p.Next(from)
	fmt.Println(ok, next.Format(time.RFC3339))
	// Output: true 2026-12-25T00:00:00Z
}

// A rejected pattern carries one of the four stable sentinels, matchable with
// errors.Is; Code maps it to the cross-implementation string code.
func ExampleCode() {
	_, err := isnow.Parse("25:00")
	fmt.Println(errors.Is(err, isnow.ErrRange), isnow.Code(err))
	// Output: true range
}
