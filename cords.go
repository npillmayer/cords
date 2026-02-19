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

// Cord stores immutable text fragments in a persistent summarized B+ tree.
//
// A cord created by
//
//	Cord{}
//
// is a valid object and behaves like the empty string.
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
	tree *btree.Tree[chunk.Chunk, chunk.Summary]
}

// FromString creates a cord from a Go string.
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

// String returns the cord as a Go string. This may be an expensive operation,
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

// IsVoid returns true if cord is "".
func (cord Cord) IsVoid() bool {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.IsVoid: cannot materialize tree")
	return tree == nil || tree.IsEmpty()
}

// Len returns the length in bytes of a cord.
func (cord Cord) Len() uint64 {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.Len: cannot materialize tree")
	if tree == nil {
		return 0
	}
	return tree.Summary().Bytes
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

// RangeLeaf returns an iterator over all current leaves as Leaf adapters.
func (cord Cord) RangeLeaf() iter.Seq[Leaf] {
	return func(yield func(Leaf) bool) {
		tree, err := treeFromCord(cord)
		assert(err == nil, "cord.RangeLeaf: cannot materialize tree")
		if tree == nil {
			return
		}
		tree.ForEachItem(func(c chunk.Chunk) bool {
			return yield(chunkLeaf{chunk: c})
		})
	}
}

// EachLeaf iterates over all leaf nodes of the cord.
func (cord Cord) EachLeaf(f func(Leaf, uint64) error) error {
	tree, convErr := treeFromCord(cord)
	assert(convErr == nil, "cord.EachLeaf: cannot materialize tree")
	if tree == nil {
		return nil
	}
	var err error
	var pos uint64
	tree.ForEachItem(func(c chunk.Chunk) bool {
		if err != nil {
			return false
		}
		leaf := chunkLeaf{chunk: c}
		err = f(leaf, pos)
		pos += leaf.Weight()
		return err == nil
	})
	return err
}

// Leaf is an interface type for leaves of a cord structure.
// Leafs carry fragments of text.
type Leaf interface {
	Weight() uint64                  // length of the leaf fragment in bytes
	String() string                  // produce the leaf fragment as a string
	Substring(uint64, uint64) []byte // substring [i:j]
	Split(uint64) (Leaf, Leaf)       // split into 2 leafs at position i
}

// StringLeaf is the default implementation of type Leaf.
// Calls to cords.FromString(…) will produce chunks rather than StringLeaf,
// but StringLeaf stays useful for compatibility constructors and tests.
type StringLeaf string

// Weight of a leaf is its string length in bytes.
func (lstr StringLeaf) Weight() uint64 {
	return uint64(len(lstr))
}

func (lstr StringLeaf) String() string {
	return string(lstr)
}

// Split splits a leaf at position i, resulting in 2 new leafs.
func (lstr StringLeaf) Split(i uint64) (Leaf, Leaf) {
	left := lstr[:i]
	right := lstr[i:]
	return left, right
}

// Substring returns a string segment of the leaf's text fragment.
func (lstr StringLeaf) Substring(i, j uint64) []byte {
	return []byte(lstr)[i:j]
}

var _ Leaf = StringLeaf("")
