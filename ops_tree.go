package cords

import (
	"fmt"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
)

// newChunkTree creates an empty rope tree with the chunk summary monoid config.
func newChunkTree() (*btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], error) {
	cfg := btree.Config[chunk.Chunk, chunk.Summary, btree.NO_EXT]{Monoid: chunk.Monoid{}}
	return btree.New[chunk.Chunk, chunk.Summary](cfg)
}

// treeFromCord returns the tree representation for a cord.
//
// In the current architecture Cord is always tree-backed; this helper centralizes
// that access and keeps call sites uniform.
func treeFromCord(cord Cord) (*btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], error) {
	if cord.tree != nil {
		return cord.tree, nil
	}
	return newChunkTree()
}

// cordFromTree wraps a tree as a Cord, normalizing empty trees to Cord{}.
func cordFromTree(tree *btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT]) Cord {
	if tree == nil || tree.IsEmpty() {
		return Cord{}
	}
	return Cord{tree: tree}
}

// concatTree concatenates one or more cords while ignoring empty operands.
func concatTree(cord Cord, others ...Cord) Cord {
	all := make([]Cord, 0, len(others)+1)
	if !cord.IsVoid() {
		all = append(all, cord)
	}
	for _, c := range others {
		if !c.IsVoid() {
			all = append(all, c)
		}
	}
	if len(all) == 0 {
		return Cord{}
	}
	base, err := treeFromCord(all[0])
	assert(err == nil, "concatTree: cannot convert base cord to tree")
	for _, c := range all[1:] {
		other, convErr := treeFromCord(c)
		assert(convErr == nil, "concatTree: cannot convert rhs cord to tree")
		base, err = base.Concat(other)
		assert(err == nil, "concatTree: btree concat failed")
	}
	return cordFromTree(base)
}

// splitTree splits a cord at byte offset i.
func splitTree(cord Cord, i uint64) (Cord, Cord, error) {
	tree, err := treeFromCord(cord)
	if err != nil {
		return Cord{}, Cord{}, err
	}
	left, right, err := splitTreeByByte(tree, i)
	if err != nil {
		return Cord{}, Cord{}, err
	}
	return cordFromTree(left), cordFromTree(right), nil
}

// splitTreeByByte splits a tree at byte offset i.
//
// If i lands in the middle of a chunk, that chunk is split first (UTF-8 boundary
// required), then both chunk parts are inserted on the corresponding sides.
func splitTreeByByte(tree *btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], i uint64) (*btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], *btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], error) {
	total := tree.Summary().Bytes
	if i > total {
		return nil, nil, ErrIndexOutOfBounds
	}
	if i == 0 {
		empty, err := newChunkTree()
		if err != nil {
			return nil, nil, err
		}
		return empty, tree, nil
	}
	if i == total {
		empty, err := newChunkTree()
		if err != nil {
			return nil, nil, err
		}
		return tree, empty, nil
	}
	cursor, err := btree.NewCursor[chunk.Chunk, chunk.Summary, btree.NO_EXT, uint64](tree, chunk.ByteDimension{})
	if err != nil {
		return nil, nil, err
	}
	itemIndex, acc, err := cursor.Seek(i)
	if err != nil {
		return nil, nil, err
	}
	if itemIndex < 0 || itemIndex >= tree.Len() {
		return nil, nil, ErrIndexOutOfBounds
	}
	item, err := tree.At(itemIndex)
	if err != nil {
		return nil, nil, err
	}
	itemBytes := item.Summary().Bytes
	before := acc - itemBytes
	local := i - before
	if local == 0 {
		l, r, err := tree.SplitAt(itemIndex)
		return l, r, err
	}
	if local == itemBytes {
		l, r, err := tree.SplitAt(itemIndex + 1)
		return l, r, err
	}
	leftSlice, rightSlice, err := item.SplitAt(int(local))
	if err != nil {
		return nil, nil, fmt.Errorf("split index %d is not on UTF-8 boundary: %w", i, err)
	}
	leftChunk, err := chunk.NewBytes(leftSlice.Bytes())
	if err != nil {
		return nil, nil, err
	}
	rightChunk, err := chunk.NewBytes(rightSlice.Bytes())
	if err != nil {
		return nil, nil, err
	}
	left, right, err := tree.SplitAt(itemIndex)
	if err != nil {
		return nil, nil, err
	}
	right, err = right.DeleteAt(0)
	if err != nil {
		return nil, nil, err
	}
	left, err = left.InsertAt(left.Len(), leftChunk)
	if err != nil {
		return nil, nil, err
	}
	right, err = right.InsertAt(0, rightChunk)
	if err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

// insertTree inserts c into cord at byte offset i.
func insertTree(cord Cord, c Cord, i uint64) (Cord, error) {
	if cord.IsVoid() && i == 0 {
		return c, nil
	}
	if cord.Len() < i {
		return Cord{}, ErrIndexOutOfBounds
	}
	if c.IsVoid() {
		return cord, nil
	}
	left, right, err := splitTree(cord, i)
	if err != nil {
		return Cord{}, err
	}
	return concatTree(left, c, right), nil
}

// cutTree removes byte range [i, i+l) and returns (remaining, removed).
func cutTree(cord Cord, i, l uint64) (Cord, Cord, error) {
	if l == 0 {
		return cord, Cord{}, nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return Cord{}, Cord{}, ErrIndexOutOfBounds
	}
	left, rest, err := splitTree(cord, i)
	if err != nil {
		return Cord{}, Cord{}, err
	}
	mid, right, err := splitTree(rest, l)
	if err != nil {
		return Cord{}, Cord{}, err
	}
	return concatTree(left, right), mid, nil
}

// substrTree returns a new cord for byte range [i, i+l).
func substrTree(cord Cord, i, l uint64) (Cord, error) {
	if l == 0 {
		return Cord{}, nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return Cord{}, ErrIndexOutOfBounds
	}
	_, rest, err := splitTree(cord, i)
	if err != nil {
		return Cord{}, err
	}
	sub, _, err := splitTree(rest, l)
	if err != nil {
		return Cord{}, err
	}
	return sub, nil
}

// reportTree materializes byte range [i, i+l) as a Go string.
func reportTree(cord Cord, i, l uint64) (string, error) {
	sub, err := substrTree(cord, i, l)
	if err != nil {
		return "", err
	}
	return sub.String(), nil
}

// indexTree returns the chunk containing byte i and the local offset in that chunk.
func indexTree(cord Cord, i uint64) (chunk.Chunk, uint64, error) {
	tree, err := treeFromCord(cord)
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	if i >= tree.Summary().Bytes {
		return chunk.Chunk{}, 0, ErrIndexOutOfBounds
	}
	cursor, err := btree.NewCursor[chunk.Chunk, chunk.Summary, btree.NO_EXT, uint64](tree, chunk.ByteDimension{})
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	itemIndex, acc, err := cursor.Seek(i + 1)
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	item, err := tree.At(itemIndex)
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	before := acc - item.Summary().Bytes
	return item, i - before, nil
}
