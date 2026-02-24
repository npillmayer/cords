package btree

import (
	"bytes"
	"errors"
	"testing"

	"github.com/npillmayer/cords/chunk"
)

func makeTextTree(t *testing.T) *Tree[textChunk, textSummary, NO_EXT] {
	t.Helper()
	tree, err := New[textChunk, textSummary](Config[textChunk, textSummary, NO_EXT]{
		Monoid: textMonoid{},
	})
	if err != nil {
		t.Fatalf("failed to create tree: %v", err)
	}
	return tree
}

func chunks(strs ...string) []textChunk {
	out := make([]textChunk, 0, len(strs))
	for _, s := range strs {
		out = append(out, fromString(s))
	}
	return out
}

func chunkStrings(items []textChunk) []string {
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
	cloned.items[1] = fromString("XX")
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
	l2.items = append(l2.items, fromString("f\n"))
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
	left, right, err := tree.insertIntoLeafLocal(leaf, 1, fromString("X"))
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
	base := make([]textChunk, 0, MaxChildren)
	for i := range MaxChildren {
		base = append(base, fromString(string(rune('a'+(i%26)))))
	}
	leaf := tree.makeLeaf(base)
	left, right, err := tree.insertIntoLeafLocal(leaf, MaxChildren/2, fromString("X"))
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

// --- Text-chunks as items --------------------------------------------------

// textSummary is a default summary type for text chunks.
//
// It is intentionally small and additive so it can serve as a base for
// dimensioned cursor operations.
type textSummary struct {
	Bytes uint64
	Lines uint64
}

// textChunk is a rope leaf item that is summarized at the type level.
type textChunk []byte

// fromString creates a text chunk from a Go string.
func fromString(s string) textChunk {
	return textChunk([]byte(s))
}

func (chunk textChunk) String() string {
	return string(chunk)
}

// Summary returns bytes/lines for this chunk.
func (chunk textChunk) Summary() textSummary {
	return textSummary{
		Bytes: uint64(len(chunk)),
		Lines: uint64(bytes.Count(chunk, []byte{'\n'})),
	}
}

// textMonoid aggregates textSummary values.
type textMonoid struct{}

// Zero returns the neutral summary.
func (textMonoid) Zero() textSummary {
	return textSummary{}
}

// Add combines two summaries.
func (textMonoid) Add(left, right textSummary) textSummary {
	return textSummary{
		Bytes: left.Bytes + right.Bytes,
		Lines: left.Lines + right.Lines,
	}
}

// byteDimension seeks/accumulates by byte count.
type byteDimension struct{}

// Zero returns 0 bytes.
func (byteDimension) Zero() uint64 { return 0 }

// Add adds bytes from summary into accumulator.
func (byteDimension) Add(acc uint64, summary textSummary) uint64 {
	return acc + summary.Bytes
}

// Compare compares dimension progress to target.
func (byteDimension) Compare(acc uint64, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}

// lineDimension seeks/accumulates by newline count.
type lineDimension struct{}

// Zero returns 0 lines.
func (lineDimension) Zero() uint64 { return 0 }

// Add adds lines from summary into accumulator.
func (lineDimension) Add(acc uint64, summary textSummary) uint64 {
	return acc + summary.Lines
}

// Compare compares dimension progress to target.
func (lineDimension) Compare(acc uint64, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------

func mustChunk(t *testing.T, s string) chunk.Chunk {
	t.Helper()
	c, err := chunk.New(s)
	if err != nil {
		t.Fatalf("chunk.New(%q) failed: %v", s, err)
	}
	return c
}

func TestTreeWithChunkItemsAndSummaryDimensions(t *testing.T) {
	tree, err := New[chunk.Chunk, chunk.Summary](Config[chunk.Chunk, chunk.Summary, NO_EXT]{
		Monoid: chunk.Monoid{},
	})
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}

	tree, err = tree.InsertAt(0,
		mustChunk(t, "ab"),
		mustChunk(t, "😀\n"),
		mustChunk(t, "x"),
	)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	sum := tree.Summary()
	if sum.Bytes != 8 || sum.Chars != 5 || sum.Lines != 1 {
		t.Fatalf("unexpected tree summary: %+v", sum)
	}

	byteCur, err := NewCursor[chunk.Chunk, chunk.Summary, NO_EXT, uint64](tree, chunk.ByteDimension{})
	if err != nil {
		t.Fatalf("byte cursor create failed: %v", err)
	}
	idx, acc, err := byteCur.Seek(3)
	if err != nil {
		t.Fatalf("byte seek failed: %v", err)
	}
	if idx != 1 || acc != 7 {
		t.Fatalf("unexpected byte seek result idx=%d acc=%d", idx, acc)
	}

	charCur, err := NewCursor[chunk.Chunk, chunk.Summary, NO_EXT, uint64](tree, chunk.CharDimension{})
	if err != nil {
		t.Fatalf("char cursor create failed: %v", err)
	}
	idx, acc, err = charCur.Seek(3)
	if err != nil {
		t.Fatalf("char seek failed: %v", err)
	}
	if idx != 1 || acc != 4 {
		t.Fatalf("unexpected char seek result idx=%d acc=%d", idx, acc)
	}

	lineCur, err := NewCursor[chunk.Chunk, chunk.Summary, NO_EXT, uint64](tree, chunk.LineDimension{})
	if err != nil {
		t.Fatalf("line cursor create failed: %v", err)
	}
	idx, acc, err = lineCur.Seek(1)
	if err != nil {
		t.Fatalf("line seek failed: %v", err)
	}
	if idx != 1 || acc != 1 {
		t.Fatalf("unexpected line seek result idx=%d acc=%d", idx, acc)
	}
}

func TestPrefixSummaryWithChunkItems(t *testing.T) {
	tree, err := New[chunk.Chunk, chunk.Summary](Config[chunk.Chunk, chunk.Summary, NO_EXT]{
		Monoid: chunk.Monoid{},
	})
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	tree, err = tree.InsertAt(0,
		mustChunk(t, "ab"),
		mustChunk(t, "😀\n"),
		mustChunk(t, "x"),
	)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	s0, err := tree.PrefixSummary(0)
	if err != nil {
		t.Fatalf("PrefixSummary(0) failed: %v", err)
	}
	if s0 != (chunk.Summary{}) {
		t.Fatalf("unexpected prefix summary at 0: %+v", s0)
	}

	s1, err := tree.PrefixSummary(1)
	if err != nil {
		t.Fatalf("PrefixSummary(1) failed: %v", err)
	}
	if s1.Bytes != 2 || s1.Chars != 2 || s1.Lines != 0 {
		t.Fatalf("unexpected prefix summary at 1: %+v", s1)
	}

	s2, err := tree.PrefixSummary(2)
	if err != nil {
		t.Fatalf("PrefixSummary(2) failed: %v", err)
	}
	if s2.Bytes != 7 || s2.Chars != 4 || s2.Lines != 1 {
		t.Fatalf("unexpected prefix summary at 2: %+v", s2)
	}

	s3, err := tree.PrefixSummary(3)
	if err != nil {
		t.Fatalf("PrefixSummary(3) failed: %v", err)
	}
	if s3 != tree.Summary() {
		t.Fatalf("prefix summary at Len() should equal full summary: got %+v want %+v", s3, tree.Summary())
	}

	_, err = tree.PrefixSummary(4)
	if !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for PrefixSummary(4), got %v", err)
	}
}
