package btree

import "testing"

func BenchmarkInsertAtStub(b *testing.B) {
	tree, err := New[TextChunk, TextSummary](Config[TextChunk, TextSummary, NO_EXT]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}
	_ = tree
	b.Skip("benchmark scaffold: InsertAt is not implemented yet")
}
