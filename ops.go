package cords

import (
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
)

// Concat concatenates cords and returns a new cord.
//
// Inputs are not modified; unchanged tree parts are shared structurally.
func Concat(cord Cord, others ...Cord) Cord {
	x := toCordext(cord)
	xOthers := make([]cordext.CordEx[btree.NO_EXT], 0, len(others))
	for _, other := range others {
		xOthers = append(xOthers, toCordext(other))
	}
	joined, err := x.Concat(xOthers...)
	assert(err == nil, "Concat: internal inconsistency")
	return fromCordext(joined)
}

// Insert inserts a substring-cord c into cord at index i, resulting in a
// new cord. If i is greater than the length of cord, an out-of-bounds error
// is returned. Index i is a byte offset.
func Insert(cord Cord, c Cord, i uint64) (Cord, error) {
	out, err := toCordext(cord).Insert(toCordext(c), i)
	return fromCordext(out), fromCordextError(err)
}

// Split splits a cord into two new (smaller) cords right before position i.
// Position i is a byte offset.
func Split(cord Cord, i uint64) (Cord, Cord, error) {
	left, right, err := toCordext(cord).Split(i)
	return fromCordext(left), fromCordext(right), fromCordextError(err)
}

// Cut cuts out a byte range [i, i+l) from a cord. It returns a new cord
// without the cut-out segment and the cut segment itself.
func Cut(cord Cord, i, l uint64) (Cord, Cord, error) {
	remaining, cut, err := toCordext(cord).Cut(i, l)
	return fromCordext(remaining), fromCordext(cut), fromCordextError(err)
}

// Report materializes l bytes starting at byte offset i as a Go string.
func (cord Cord) Report(i, l uint64) (string, error) {
	s, err := toCordext(cord).Report(i, l)
	return s, fromCordextError(err)
}

// Substr returns a new cord representing byte range [i, i+l).
func Substr(cord Cord, i, l uint64) (Cord, error) {
	sub, err := toCordext(cord).Substr(i, l)
	return fromCordext(sub), fromCordextError(err)
}

// Index returns the cord chunk that includes byte position i, together with an
// index within that chunk.
func (cord Cord) Index(i uint64) (chunk.Chunk, uint64, error) {
	c, off, err := toCordext(cord).Index(i)
	return c, off, fromCordextError(err)
}

// FragmentCount returns the number of chunks currently stored in the cord.
func (cord Cord) FragmentCount() int {
	cnt := 0
	_ = cord.EachChunk(func(chunk.Chunk, uint64) error {
		cnt++
		return nil
	})
	return cnt
}
