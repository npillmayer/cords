package btree

import (
	"strings"
	"testing"
)

func TestFixedBackendDetectsLeafOccupancyDrift(t *testing.T) {
	tree := makeTextTree(t)
	leaf := tree.makeLeaf(chunks("a", "b"))
	tree.root = leaf
	tree.height = 1

	leaf.n = 1 // corrupt logical length on purpose

	err := tree.Check()
	if err == nil {
		t.Fatalf("expected invariant error for leaf occupancy drift")
	}
	if !strings.Contains(err.Error(), "leaf occupancy mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFixedBackendDetectsChildViewDrift(t *testing.T) {
	tree := makeTextTree(t)
	leaf := tree.makeLeaf(chunks("a"))
	inner := tree.makeInternal(leaf)
	tree.root = inner
	tree.height = 2

	// Break fixed-storage backing invariant intentionally.
	inner.children = append([]treeNode[TextChunk, TextSummary, NO_EXT](nil), inner.children...)

	err := tree.Check()
	if err == nil {
		t.Fatalf("expected invariant error for child view drift")
	}
	if !strings.Contains(err.Error(), "child view cap mismatch") && !strings.Contains(err.Error(), "child view is not backed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
