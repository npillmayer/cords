package cords

import "github.com/npillmayer/cords/chunk"

// Concat concatenates cords and returns a new cord.
func Concat(cord Cord, others ...Cord) Cord {
	return concatTree(cord, others...)
}

// Insert inserts a substring-cord c into cord at index i, resulting in a
// new cord. If i is greater than the length of cord, an out-of-bounds error
// is returned.
func Insert(cord Cord, c Cord, i uint64) (Cord, error) {
	return insertTree(cord, c, i)
}

// Split splits a cord into two new (smaller) cords right before position i.
// Split(C,i) => split C into C1 and C2, with C1=b0,...,bi-1 and C2=bi,...,bn.
func Split(cord Cord, i uint64) (Cord, Cord, error) {
	return splitTree(cord, i)
}

// Cut cuts out a substring [i...i+l) from a cord. It returns a new cord
// without the cut-out segment and the cut segment itself.
func Cut(cord Cord, i, l uint64) (Cord, Cord, error) {
	return cutTree(cord, i, l)
}

// Report outputs a substring: Report(i,l) => outputs the string bi,...,bi+l-1.
func (cord Cord) Report(i, l uint64) (string, error) {
	return reportTree(cord, i, l)
}

// Substr creates a new cord from a subset of cord.
func Substr(cord Cord, i, l uint64) (Cord, error) {
	return substrTree(cord, i, l)
}

// Index returns the cord chunk that includes byte position i, together with an
// index position within that chunk.
func (cord Cord) Index(i uint64) (chunk.Chunk, uint64, error) {
	return indexTree(cord, i)
}

// FragmentCount returns the number of fragments this cord is internally split into.
func (cord Cord) FragmentCount() int {
	cnt := 0
	_ = cord.EachChunk(func(chunk.Chunk, uint64) error {
		cnt++
		return nil
	})
	return cnt
}
