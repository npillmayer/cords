package btree

import "testing"

func makeTextTree(t *testing.T) *Tree[TextChunk, TextSummary] {
	t.Helper()
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("failed to create tree: %v", err)
	}
	return tree
}

func chunks(strs ...string) []TextChunk {
	out := make([]TextChunk, 0, len(strs))
	for _, s := range strs {
		out = append(out, FromString(s))
	}
	return out
}

func chunkStrings(items []TextChunk) []string {
	out := make([]string, 0, len(items))
	for _, c := range items {
		out = append(out, string(c))
	}
	return out
}

func TestCloneLeafCreatesIndependentSlice(t *testing.T) {
	tree := makeTextTree(t)
	leaf := tree.makeLeaf(chunks("aa", "bb", "cc"))
	cloned := tree.cloneLeaf(leaf)
	if cloned == leaf {
		t.Fatalf("cloneLeaf returned same pointer")
	}
	cloned.items[1] = FromString("XX")
	tree.recomputeLeafSummary(cloned)
	if string(leaf.items[1]) != "bb" {
		t.Fatalf("original leaf changed after clone mutation")
	}
}

func TestRecomputeInnerSummary(t *testing.T) {
	tree := makeTextTree(t)
	l1 := tree.makeLeaf(chunks("ab", "c\n"))
	l2 := tree.makeLeaf(chunks("de"))
	inner := tree.makeInternal(l1, l2)
	if inner.summary.Bytes != 6 || inner.summary.Lines != 1 {
		t.Fatalf("unexpected initial inner summary: %+v", inner.summary)
	}
	l2.items = append(l2.items, FromString("f\n"))
	tree.recomputeLeafSummary(l2)
	tree.recomputeInnerSummary(inner)
	if inner.summary.Bytes != 8 || inner.summary.Lines != 2 {
		t.Fatalf("unexpected recomputed inner summary: %+v", inner.summary)
	}
}

func TestInsertRemoveSliceHelpers(t *testing.T) {
	base := []int{1, 2, 3, 4}
	ins := insertAt(base, 2, 8, 9)
	wantIns := []int{1, 2, 8, 9, 3, 4}
	for i := range wantIns {
		if ins[i] != wantIns[i] {
			t.Fatalf("insertAt mismatch at %d: got %v want %v", i, ins, wantIns)
		}
	}
	rem := removeRange(ins, 1, 4)
	wantRem := []int{1, 3, 4}
	for i := range wantRem {
		if rem[i] != wantRem[i] {
			t.Fatalf("removeRange mismatch at %d: got %v want %v", i, rem, wantRem)
		}
	}
}

func TestInsertRemoveChildHelpers(t *testing.T) {
	tree := makeTextTree(t)
	l1 := tree.makeLeaf(chunks("a"))
	l2 := tree.makeLeaf(chunks("bb"))
	l3 := tree.makeLeaf(chunks("ccc"))
	inner := tree.makeInternal(l1, l3)
	tree.insertChildAt(inner, 1, l2)
	if len(inner.children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(inner.children))
	}
	if inner.summary.Bytes != 6 {
		t.Fatalf("unexpected summary after insertChildAt: %+v", inner.summary)
	}
	tree.removeChildAt(inner, 0)
	if len(inner.children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(inner.children))
	}
	if inner.summary.Bytes != 5 {
		t.Fatalf("unexpected summary after removeChildAt: %+v", inner.summary)
	}
}

func TestLeafInsertLocalNoSplit(t *testing.T) {
	tree := makeTextTree(t)
	leaf := tree.makeLeaf(chunks("a", "b", "c"))
	left, right, err := tree.insertIntoLeafLocal(leaf, 1, FromString("X"))
	if err != nil {
		t.Fatalf("insertIntoLeafLocal failed: %v", err)
	}
	if right != nil {
		t.Fatalf("unexpected split sibling for non-overflow insert")
	}
	got := chunkStrings(left.items)
	want := []string{"a", "X", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("insert order mismatch: got %v want %v", got, want)
		}
	}
	orig := chunkStrings(leaf.items)
	if len(orig) != 3 || orig[1] != "b" {
		t.Fatalf("original leaf modified unexpectedly: %v", orig)
	}
}

func TestLeafInsertLocalSplit(t *testing.T) {
	tree := makeTextTree(t)
	base := make([]TextChunk, 0, MaxChildren)
	for i := range MaxChildren {
		base = append(base, FromString(string(rune('a'+(i%26)))))
	}
	leaf := tree.makeLeaf(base)
	left, right, err := tree.insertIntoLeafLocal(leaf, MaxChildren/2, FromString("X"))
	if err != nil {
		t.Fatalf("insertIntoLeafLocal failed: %v", err)
	}
	if right == nil {
		t.Fatalf("expected split sibling, got nil")
	}
	if tree.leafOverflow(left) || tree.leafOverflow(right) {
		t.Fatalf("split result still overflows")
	}
	got := append(chunkStrings(left.items), chunkStrings(right.items)...)
	if len(got) != MaxChildren+1 {
		t.Fatalf("unexpected split output length: got %d want %d", len(got), MaxChildren+1)
	}
}
