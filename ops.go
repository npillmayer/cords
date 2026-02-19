package cords

import "github.com/npillmayer/cords/chunk"

// Concat concatenates cords and returns a new cord.
//
// Inputs are not modified; unchanged tree parts are shared structurally.
func Concat(cord Cord, others ...Cord) Cord {
	return concatTree(cord, others...)
}

// Insert inserts a substring-cord c into cord at index i, resulting in a
// new cord. If i is greater than the length of cord, an out-of-bounds error
// is returned. Index i is a byte offset.
func Insert(cord Cord, c Cord, i uint64) (Cord, error) {
	return insertTree(cord, c, i)
}

// Split splits a cord into two new (smaller) cords right before position i.
// Position i is a byte offset.
func Split(cord Cord, i uint64) (Cord, Cord, error) {
	return splitTree(cord, i)
}

// Cut cuts out a byte range [i, i+l) from a cord. It returns a new cord
// without the cut-out segment and the cut segment itself.
func Cut(cord Cord, i, l uint64) (Cord, Cord, error) {
	return cutTree(cord, i, l)
}

// Report materializes l bytes starting at byte offset i as a Go string.
func (cord Cord) Report(i, l uint64) (string, error) {
	return reportTree(cord, i, l)
}

// Substr returns a new cord representing byte range [i, i+l).
func Substr(cord Cord, i, l uint64) (Cord, error) {
	return substrTree(cord, i, l)
}

// Index returns the cord chunk that includes byte position i, together with an
// index within that chunk.
func (cord Cord) Index(i uint64) (chunk.Chunk, uint64, error) {
	return indexTree(cord, i)
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
