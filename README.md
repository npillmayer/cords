# Cords
Cords of text as a versatile string enhancement

## Status

Work in progress, please be patient!

----------

From a paper by Hans-J. Boehm, Russ Atkinson and Michael Plass, 1995:

### What's wrong with Strings?

Programming languages such as C and traditional Pascal provide a built-in notion
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

Strings represented as contiguous arrays of characters, as in C or Pascal, violate most of these.

From:

### Ropes: an Alternative to Strings

1995, by

hans-j. boehm, russ atkinson and michael plass

Xerox PARC, 3333 Coyote Hill Rd., Palo Alto, CA 94304, U.S.A. (email:boehm@parc.xerox.com)

## Other / similar Solutions

Raph Levien implemented a “Ropes” data type for his Xi editor in Rust:
[Blog entry on Raph's blog](https://raphlinus.github.io/xi/2020/06/27/xi-retrospective.html)
