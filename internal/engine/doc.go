// Package engine implements the isnow pattern engine: recognition of pattern
// source via the grammar repo's ANTLR-generated parser, the shorthand-ladder
// role assignment, compilation of every field term to a predicate, membership
// (Holds) over a broken-down instant, and occurrence derivation (Next/Prev)
// under the 100-year search horizon.
//
// The root go-isnow package is a thin facade over this package: everything
// exported here is re-exported there unchanged, so external callers never
// import the engine directly. Errors are the internal/constants sentinels,
// matchable with errors.Is.
package engine
