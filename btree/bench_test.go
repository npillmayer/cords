package btree

import "testing"

func BenchmarkInsertAtStub(b *testing.B) {
	tree, err := New[textChunk, textSummary](Config[textChunk, textSummary, NO_EXT]{
		Monoid: textMonoid{},
	})
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}
	_ = tree
	b.Skip("benchmark scaffold: InsertAt is not implemented yet")
}
