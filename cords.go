package cords

/*
BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer

Please refer to the License file in the repository root.

*/

import (
	"bytes"
	"iter"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
)

// Cord stores immutable UTF-8 text fragments in a persistent summarized B+ tree.
//
// A cord created by
//
//	Cord{}
//
// is a valid object and behaves like the empty string.
//
// Methods that take or return positions use byte offsets.
//
// Due to their internal structure cords do have performance characteristics
// differing from Go strings or byte arrays.
//
//	Operation     |   Rope          |  String
//	--------------+-----------------+--------
//	Index         |   O(log n)      |   O(1)
//	Split         |   O(log n)      |   O(1)
//	Iterate       |   O(n)          |   O(n)
//
//	Concatenate   |   O(log n)      |   O(n)
//	Insert        |   O(log n)      |   O(n)
//	Delete        |   O(log n)      |   O(n)
//
// For use cases with many editing operations on large texts, cords have stable
// performance and space characteristics.
type Cord struct {
	tree *btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT]
}

// FromString creates a cord from a Go string.
//
// The input string must be valid UTF-8. Invalid input triggers an internal
// assertion panic, matching package invariants for stored text.
func FromString(s string) Cord {
	parts, err := splitToChunks([]byte(s))
	assert(err == nil, "FromString requires valid UTF-8 input")
	tree, e := newChunkTree()
	assert(e == nil, "FromString: cannot create chunk tree")
	if len(parts) > 0 {
		tree, e = tree.InsertAt(0, parts...)
		assert(e == nil, "FromString: cannot insert chunks")
	}
	return cordFromTree(tree)
}

// String returns the complete cord as a Go string. This may be an expensive operation,
// as it will allocate a buffer for all the bytes of the cord and collect all
// fragments to a single continuous string.
func (cord Cord) String() string {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.String: cannot materialize tree")
	if tree == nil || tree.IsEmpty() {
		return ""
	}
	var bf bytes.Buffer
	tree.ForEachItem(func(c chunk.Chunk) bool {
		_, _ = bf.WriteString(c.String())
		return true
	})
	return bf.String()
}

// IsVoid reports whether the cord has no bytes.
func (cord Cord) IsVoid() bool {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.IsVoid: cannot materialize tree")
	return tree == nil || tree.IsEmpty()
}

// Len returns the cord length in bytes.
func (cord Cord) Len() uint64 {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.Len: cannot materialize tree")
	if tree == nil {
		return 0
	}
	return tree.Summary().Bytes
}

// Summary returns aggregate byte/rune/line counts for the cord.
func (cord Cord) Summary() chunk.Summary {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.Summary: cannot materialize tree")
	if tree == nil {
		return chunk.Summary{}
	}
	return tree.Summary()
}

// CharCount returns the number of UTF-8 runes in the cord.
func (cord Cord) CharCount() uint64 {
	return cord.Summary().Chars
}

// LineCount returns the number of newline characters in the cord.
func (cord Cord) LineCount() uint64 {
	return cord.Summary().Lines
}

// height returns the total height of a cord's tree.
func (cord Cord) height() int {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.height: cannot materialize tree")
	if tree == nil || tree.IsEmpty() {
		return 0
	}
	return tree.Height()
}

// RangeChunk returns an iterator over all chunks in logical order.
func (cord Cord) RangeChunk() iter.Seq[chunk.Chunk] {
	return func(yield func(chunk.Chunk) bool) {
		for seg := range cord.RangeTextSegment() {
			if !yield(seg.Chunk()) {
				return
			}
		}
	}
}

// RangeTextSegment returns an iterator over all text segments in logical order.
func (cord Cord) RangeTextSegment() iter.Seq[TextSegment] {
	return func(yield func(TextSegment) bool) {
		tree, err := treeFromCord(cord)
		assert(err == nil, "cord.RangeTextSegment: cannot materialize tree")
		if tree == nil {
			return
		}
		tree.ForEachItem(func(c chunk.Chunk) bool {
			return yield(newTextSegment(c))
		})
	}
}

// EachChunk visits all chunks in logical order.
//
// The callback receives each chunk and its starting byte offset. Iteration stops
// at the first callback error and returns that error to the caller.
func (cord Cord) EachChunk(f func(chunk.Chunk, uint64) error) error {
	return cord.EachTextSegment(func(seg TextSegment, pos uint64) error {
		return f(seg.Chunk(), pos)
	})
}

// EachTextSegment visits all text segments in logical order.
//
// The callback receives each segment and its starting byte offset. Iteration
// stops at the first callback error and returns that error to the caller.
func (cord Cord) EachTextSegment(f func(TextSegment, uint64) error) error {
	tree, convErr := treeFromCord(cord)
	assert(convErr == nil, "cord.EachTextSegment: cannot materialize tree")
	if tree == nil {
		return nil
	}
	var err error
	var pos uint64
	tree.ForEachItem(func(c chunk.Chunk) bool {
		if err != nil {
			return false
		}
		err = f(newTextSegment(c), pos)
		pos += c.Summary().Bytes
		return err == nil
	})
	return err
}
