package cords

import (
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
)

// newChunkTree creates an empty rope tree with the chunk summary monoid config.
//
// This is a low-level helper kept for core Cord methods and bridge conversion.
func newChunkTree() (*btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], error) {
	cfg := btree.Config[chunk.Chunk, chunk.Summary, btree.NO_EXT]{Monoid: chunk.Monoid{}}
	return btree.New[chunk.Chunk, chunk.Summary](cfg)
}

// treeFromCord returns the tree representation for a cord.
//
// In the current architecture Cord is tree-backed; this helper centralizes
// access and materializes an empty tree for zero-value cords.
func treeFromCord(cord Cord) (*btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], error) {
	if cord.tree != nil {
		return cord.tree, nil
	}
	tree, err := newChunkTree()
	if err != nil {
		return nil, err
	}
	cord.tree = tree
	return tree, nil
}

// cordFromTree wraps a tree as a Cord, normalizing empty trees to Cord{}.
func cordFromTree(tree *btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT]) Cord {
	if tree == nil || tree.IsEmpty() {
		return Cord{}
	}
	return Cord{tree: tree}
}
