/*
Package cords offers a versatile string enhancement to ease handling of texts.

Cords

Cords (or sometimes called ropes) organize fragments of immutable text internally
in a tree-structure. This speeds up frequent string-operations like concatenation,
especially for long strings. This package aims towards applications which have to
deal with text, i.e., large amounts of organized strings.

From Wikipedia:
In computer programming, a rope, or cord, is a data structure composed of
smaller strings that is used to efficiently store and manipulate a very long string.
For example, a text editing program may use a rope to represent the text being edited,
so that operations such as insertion, deletion, and random access can be
done efficiently. […] In summary, ropes are preferable when the data is large
and modified often.

_________________________________________________________________________

From a paper by Hans-J. Boehm, Russ Atkinson and Michael Plass, 1995:

Ropes, an Alternative to Strings

Xerox PARC, 3333 Coyote Hill Rd., Palo Alto, CA 94304, U.S.A.
(email:boehm@parc.xerox.com)

What's wrong with Strings?

Programming languages such as C […] provide a built-in notion
of strings as essentially fixed length arrays of characters. The language itself provides
the array primitives for accessing such strings, plus often a collection of library
routines for higher level operations such as string concatenation. Thus the implementation
is essentially constrained to represent strings as contiguous arrays of characters,
with or without additional space for a length, expansion room, etc. […] We desire the following
characteristics:

1. Immutable strings, i.e. strings that cannot be modified in place, should be well
supported. A procedure should be able to operate on a string it was passed
without danger of accidentally modifying the caller’s data structures. This
becomes particularly important in the presence of concurrency, where in-place
updates to strings would often have to be properly synchronized. […]

2. Commonly occurring operations on strings should be efficient. In particular (non-destructive)
concatenation of strings and non-destructive substring operations should be fast,
and should not require excessive amounts of space.

3. Common string operations should scale to long strings. There should be no practical bound
on the length of strings. Performance should remain acceptable for long strings. […]

4. It should be as easy as possible to treat some other representation of
‘sequenceof character’ (e.g. a file) as a string. Functions on strings should be maximally reusable.

Strings represented as contiguous arrays of characters, as in C or Pascal,
violate most of these.

_________________________________________________________________________

In Go, these points of critique are somewhat mitigated with slices. However,
slices will carry only so far, and cords add a layer of convenience and
stable performance characteristics on top of them. You can think of cords
as fancy slices of text, with some additional functionality.

Cords may be constructed from various sources, with the simplest case being
a call to

    cords.FromString("Hello World")

Other possibilities are cords from text files or from HTML documents.

_________________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package cords

// TODO
//
// It would probably be wise to re-base cords on an external production-ready tree implementation.
// Possible candidates would be:
// - https://github.com/google/btree
// - https://github.com/petar/gollrb
// Rust's ropes implementation rests on a B-Tree. This makes sense considering the good support
// of copy-on-write semantics in Vecs, and of course reduces the tree height for the case
// of a lot of smallish nodes. Currently I am not sure my use cases will ever fall into this
// category, but if one thinks of an interactive authoring environment, where text modifications
// arrive in high frequency, it could be the right way to go.
// On the other hand, I did not take the time to look into those libraries from a "persistent
// data structure" point of view, which is a hard requirement for me (and should be for clients
// of cords as well). In Go this kind of thinking does not come natural for authors of general
// purpose libraries, as obviously most clients of such libraries prefer space-efficient modifications
// in place. But I hope to have some spare time in the near future to evaluate btree and bollrb
// in this respect.

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'cords'.
func tracer() tracing.Trace {
	return tracing.Select("cords")
}

// CordError is an error type for the cords module
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

// ErrIllegalDelimiterPattern is flagged if a given delimiter pattern is
// either not compilable as a valid regular expression or if it accepts
// the empty string as a match.
const ErrIllegalDelimiterPattern = CordError("illegal delimiter pattern")
