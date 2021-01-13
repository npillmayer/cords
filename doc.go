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
stable performance characteristics on top of them.

Cords focus on large amounts of text. When dealing with shorter strings which do
not form an overall text, cords may add a performance penalty.

_________________________________________________________________________

BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*/
package cords

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to a global core-tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
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
