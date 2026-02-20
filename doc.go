/*
Package cords implements a persistent rope for UTF-8 text.

A Cord stores text in immutable chunks inside a summarized B+ tree. Edit-like
operations such as concat, split, cut, and insert are non-destructive: they
return new Cord values and preserve the original input cord.

All positional APIs in this package operate on byte offsets (not rune indexes).
Callers that need rune-level navigation should convert explicitly at their
application boundary.

Typical usage:

	c := cords.FromString("Hello World")
	c2, _ := cords.Insert(c, cords.FromString(","), 5)
	s, _ := c2.Report(0, c2.Len())

Extension usage:

	// ext implements cords.TextSegmentExtension[E]
	ec, _ := cords.FromStringWithExtension("Hello\nWorld\n", ext)
	total, _ := ec.Ext()
	_ = total

Extension builder usage:

	b, _ := cords.NewBuilderWithExtension(ext)
	_ = b.AppendString("Hello\n")
	_ = b.AppendString("World\n")
	ec = b.Cord()

Package `chunk` contains the text-fragment type used by this package. Package
`btree` contains the generic persistent summarized B+ tree implementation.
*/
package cords

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'cords'.
func tracer() tracing.Trace {
	return tracing.Select("cords")
}

// CordError is the package error type.
type CordError string

func (e CordError) Error() string {
	return string(e)
}

// ErrCordCompleted signals that a cord builder has already completed a cord and
// it's illegal to further add fragments.
const ErrCordCompleted = CordError("forbidden to add fragements; cord has been completed")

// ErrIndexOutOfBounds is flagged whenever a cord position is
// greater than the length of the cord.
const ErrIndexOutOfBounds = CordError("index out of bounds")

// ErrIllegalArguments is flagged whenever function parameters are invalid.
const ErrIllegalArguments = CordError("illegal arguments")

// ErrIllegalPosition signals that a Pos is inconsistent for a target cord.
const ErrIllegalPosition = CordError("illegal position")

// ErrIllegalDelimiterPattern is flagged if a given delimiter pattern is
// either not compilable as a valid regular expression or if it accepts
// the empty string as a match.
const ErrIllegalDelimiterPattern = CordError("illegal delimiter pattern")

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
