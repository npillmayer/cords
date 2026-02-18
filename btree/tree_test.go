package btree

import (
	"strconv"
	"testing"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	_, err := New[TextChunk, TextSummary](Config[TextSummary]{})
	if err == nil {
		t.Fatalf("expected invalid config error, got nil")
	}
}

func TestNewStoresMonoidConfig(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := tree.Config()
	if cfg.Monoid == nil {
		t.Fatalf("expected monoid to be set in normalized config")
	}
}

func TestCheckEmptyTree(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := tree.Check(); err != nil {
		t.Fatalf("expected empty tree to be valid, got %v", err)
	}
	if tree.Len() != 0 || tree.Height() != 0 {
		t.Fatalf("unexpected empty tree state len=%d height=%d", tree.Len(), tree.Height())
	}
}

func TestCheckManualLeafRoot(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tree.root = tree.makeLeaf([]TextChunk{
		FromString("hello"),
		FromString(" world\n"),
	})
	tree.height = 1
	if err := tree.Check(); err != nil {
		t.Fatalf("expected tree to validate, got %v", err)
	}
	if tree.Len() != 2 {
		t.Fatalf("unexpected item count: %d", tree.Len())
	}
	s := tree.Summary()
	if s.Bytes != 12 || s.Lines != 1 {
		t.Fatalf("unexpected summary: %+v", s)
	}
}

func TestCursorRequiresDimension(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = NewCursor[TextChunk, TextSummary, uint64](tree, nil)
	if err == nil {
		t.Fatalf("expected dimension error, got nil")
	}
}

func collectTextItems(tree *Tree[TextChunk, TextSummary]) []string {
	if tree == nil || tree.root == nil {
		return nil
	}
	var out []string
	var walk func(treeNode[TextChunk, TextSummary])
	walk = func(n treeNode[TextChunk, TextSummary]) {
		if n.isLeaf() {
			leaf := n.(*leafNode[TextChunk, TextSummary])
			for _, item := range leaf.items {
				out = append(out, string(item))
			}
			return
		}
		inner := n.(*innerNode[TextChunk, TextSummary])
		for _, child := range inner.children {
			walk(child)
		}
	}
	walk(tree.root)
	return out
}

func TestInsertAtNoOpClone(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clone, err := tree.InsertAt(0)
	if err != nil {
		t.Fatalf("unexpected error for no-op insert: %v", err)
	}
	if clone == tree {
		t.Fatalf("expected clone to be a distinct struct pointer")
	}
}

func TestInsertAtBuildsTreeAndPreservesOriginal(t *testing.T) {
	base, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t1, err := base.InsertAt(0, FromString("a"), FromString("b"), FromString("c"))
	if err != nil {
		t.Fatalf("unexpected insert error: %v", err)
	}
	t2, err := t1.InsertAt(1, FromString("X"))
	if err != nil {
		t.Fatalf("unexpected insert error: %v", err)
	}
	if got := collectTextItems(base); len(got) != 0 {
		t.Fatalf("base tree changed unexpectedly: %v", got)
	}
	got1 := collectTextItems(t1)
	want1 := []string{"a", "b", "c"}
	for i := range want1 {
		if got1[i] != want1[i] {
			t.Fatalf("t1 mismatch at %d: got %v want %v", i, got1, want1)
		}
	}
	got2 := collectTextItems(t2)
	want2 := []string{"a", "X", "b", "c"}
	for i := range want2 {
		if got2[i] != want2[i] {
			t.Fatalf("t2 mismatch at %d: got %v want %v", i, got2, want2)
		}
	}
	if err := t1.Check(); err != nil {
		t.Fatalf("t1 invariant check failed: %v", err)
	}
	if err := t2.Check(); err != nil {
		t.Fatalf("t2 invariant check failed: %v", err)
	}
}

func TestInsertAtRootSplitAndInternalPropagation(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With fixed degree 12, a few hundred items trigger internal split/root growth.
	for i := 0; i < 200; i++ {
		tree, err = tree.InsertAt(tree.Len(), FromString(strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}
	if tree.Height() < 3 {
		t.Fatalf("expected height >= 3 after propagated splits, got %d", tree.Height())
	}
	if err := tree.Check(); err != nil {
		t.Fatalf("invariant check failed: %v", err)
	}
	got := collectTextItems(tree)
	if len(got) != 200 {
		t.Fatalf("unexpected item count: %d", len(got))
	}
	for i := 0; i < 200; i++ {
		if got[i] != strconv.Itoa(i) {
			t.Fatalf("unexpected order at %d: got %q want %q", i, got[i], strconv.Itoa(i))
		}
	}
}

func TestSplitAtKeepsOrderAndPersistence(t *testing.T) {
	base, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 9; i++ {
		base, err = base.InsertAt(base.Len(), FromString(strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}
	left, right, err := base.SplitAt(4)
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}
	gotLeft := collectTextItems(left)
	gotRight := collectTextItems(right)
	wantLeft := []string{"0", "1", "2", "3"}
	wantRight := []string{"4", "5", "6", "7", "8"}
	for i := range wantLeft {
		if gotLeft[i] != wantLeft[i] {
			t.Fatalf("left mismatch at %d: got %v want %v", i, gotLeft, wantLeft)
		}
	}
	for i := range wantRight {
		if gotRight[i] != wantRight[i] {
			t.Fatalf("right mismatch at %d: got %v want %v", i, gotRight, wantRight)
		}
	}
	gotBase := collectTextItems(base)
	if len(gotBase) != 9 {
		t.Fatalf("base changed unexpectedly: %v", gotBase)
	}
	if err := left.Check(); err != nil {
		t.Fatalf("left invariants failed: %v", err)
	}
	if err := right.Check(); err != nil {
		t.Fatalf("right invariants failed: %v", err)
	}
}

func TestConcatKeepsInputsAndProducesCombinedOrder(t *testing.T) {
	left, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	right, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 5; i++ {
		left, err = left.InsertAt(left.Len(), FromString(strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("left insert %d failed: %v", i, err)
		}
	}
	for i := 5; i < 9; i++ {
		right, err = right.InsertAt(right.Len(), FromString(strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("right insert %d failed: %v", i, err)
		}
	}
	combined, err := left.Concat(right)
	if err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	want := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8"}
	got := collectTextItems(combined)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("combined mismatch at %d: got %v want %v", i, got, want)
		}
	}
	if len(collectTextItems(left)) != 5 || len(collectTextItems(right)) != 4 {
		t.Fatalf("input trees changed unexpectedly")
	}
	if err := combined.Check(); err != nil {
		t.Fatalf("combined invariants failed: %v", err)
	}
}

func TestSplitAtSharesUntouchedSubtree(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 20; i++ {
		tree, err = tree.InsertAt(tree.Len(), FromString(strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}
	root, ok := tree.root.(*innerNode[TextChunk, TextSummary])
	if !ok || len(root.children) < 2 {
		t.Fatalf("expected an internal root with at least 2 children")
	}
	splitIndex := tree.countItems(root.children[0]) + 1 // force split into 2nd root child
	left, _, err := tree.SplitAt(splitIndex)
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}
	leftRoot, ok := left.root.(*innerNode[TextChunk, TextSummary])
	if !ok || len(leftRoot.children) < 1 {
		t.Fatalf("expected left root to be internal with children")
	}
	if leftRoot.children[0] != root.children[0] {
		t.Fatalf("expected untouched left subtree to be shared")
	}
}

func buildTreeWithRootChildren(t *testing.T, startValue int, minRootChildren int) *Tree[TextChunk, TextSummary] {
	t.Helper()
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 10000; i++ {
		tree, err = tree.InsertAt(tree.Len(), FromString(strconv.Itoa(startValue+i)))
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
		root, ok := tree.root.(*innerNode[TextChunk, TextSummary])
		if ok && tree.Height() == 2 && len(root.children) >= minRootChildren {
			return tree
		}
	}
	t.Fatalf("failed to build tree with >=%d root children", minRootChildren)
	return nil
}

func TestConcatSharesRootsWhenJoinCannotMergeTopLevel(t *testing.T) {
	threshold := fixedBase + 1 // forces root-child sum > fixedMaxChildren after concat
	left := buildTreeWithRootChildren(t, 0, threshold)
	right := buildTreeWithRootChildren(t, 100000, threshold)

	leftRoot := left.root
	rightRoot := right.root
	combined, err := left.Concat(right)
	if err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if err := combined.Check(); err != nil {
		t.Fatalf("combined invariants failed: %v", err)
	}
	if combined.Height() != 3 {
		t.Fatalf("expected combined height 3, got %d", combined.Height())
	}
	root, ok := combined.root.(*innerNode[TextChunk, TextSummary])
	if !ok || len(root.children) != 2 {
		t.Fatalf("expected new root with two children")
	}
	if root.children[0] != leftRoot {
		t.Fatalf("expected combined root left child to share left root")
	}
	if root.children[1] != rightRoot {
		t.Fatalf("expected combined root right child to share right root")
	}
}

func TestConcatDifferentHeightsKeepsOrder(t *testing.T) {
	left := buildTreeWithRootChildren(t, 0, fixedBase+1) // height 2
	right, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"x", "y", "z"} {
		right, err = right.InsertAt(right.Len(), FromString(s))
		if err != nil {
			t.Fatalf("right insert failed: %v", err)
		}
	}
	combined, err := left.Concat(right)
	if err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if err := combined.Check(); err != nil {
		t.Fatalf("combined invariants failed: %v", err)
	}
	got := collectTextItems(combined)
	leftItems := collectTextItems(left)
	if len(got) != len(leftItems)+3 {
		t.Fatalf("unexpected combined length: %d", len(got))
	}
	for i := range leftItems {
		if got[i] != leftItems[i] {
			t.Fatalf("left prefix mismatch at %d", i)
		}
	}
	if got[len(leftItems)] != "x" || got[len(leftItems)+1] != "y" || got[len(leftItems)+2] != "z" {
		t.Fatalf("unexpected right suffix: %v", got[len(leftItems):])
	}
}

func TestSplitAtStructuralOnlyAcrossAllBoundaries(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 120; i++ {
		tree, err = tree.InsertAt(tree.Len(), FromString(strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}
	all := collectTextItems(tree)
	for split := 0; split <= len(all); split++ {
		left, right, err := tree.SplitAt(split)
		if err != nil {
			t.Fatalf("split at %d failed: %v", split, err)
		}
		gotLeft := collectTextItems(left)
		gotRight := collectTextItems(right)
		if len(gotLeft) != split {
			t.Fatalf("split %d: left length mismatch got=%d want=%d", split, len(gotLeft), split)
		}
		if len(gotRight) != len(all)-split {
			t.Fatalf("split %d: right length mismatch got=%d want=%d", split, len(gotRight), len(all)-split)
		}
		for i := 0; i < split; i++ {
			if gotLeft[i] != all[i] {
				t.Fatalf("split %d: left mismatch at %d", split, i)
			}
		}
		for i := split; i < len(all); i++ {
			if gotRight[i-split] != all[i] {
				t.Fatalf("split %d: right mismatch at %d", split, i)
			}
		}
	}
}
