//go:build !btree_fixed

package btree

import "testing"

func TestSplitLeafTooLargeForSinglePromotion(t *testing.T) {
	tree := makeTextTree(t)
	items := make([]TextChunk, 0, 2*DefaultDegree+1)
	for i := 0; i < 2*DefaultDegree+1; i++ {
		items = append(items, FromString("x"))
	}
	leaf := tree.makeLeaf(items)
	_, _, err := tree.splitLeaf(leaf)
	if err == nil {
		t.Fatalf("expected splitLeaf to fail for oversized split")
	}
}
