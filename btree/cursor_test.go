package btree

import (
	"errors"
	"testing"
)

type extBytes struct{}

func (extBytes) MagicID() string { return "ext:bytes" }
func (extBytes) Zero() uint64    { return 0 }
func (extBytes) FromItem(_ textChunk, s textSummary) uint64 {
	return s.Bytes
}
func (extBytes) Extend(_ textSummary)          {}
func (extBytes) Add(left, right uint64) uint64 { return left + right }

type Uint64Dimension struct{}

func (Uint64Dimension) Zero() uint64 { return 0 }
func (Uint64Dimension) Add(acc, summary uint64) uint64 {
	return acc + summary
}
func (Uint64Dimension) Compare(acc, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}

func TestCursorSeekBytes(t *testing.T) {
	tree, err := New(Config[textChunk, textSummary, NO_EXT]{
		Monoid: textMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), fromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	cursor, err := NewCursor(tree, byteDimension{})
	if err != nil {
		t.Fatalf("new cursor failed: %v", err)
	}

	type tc struct {
		target uint64
		idx    int
		acc    uint64
	}
	cases := []tc{
		{target: 0, idx: 0, acc: 0},
		{target: 1, idx: 0, acc: 2},
		{target: 2, idx: 0, acc: 2},
		{target: 3, idx: 1, acc: 4},
		{target: 4, idx: 1, acc: 4},
		{target: 5, idx: 2, acc: 8},
		{target: 9, idx: 3, acc: 8},
	}
	for _, c := range cases {
		idx, acc, err := cursor.Seek(c.target)
		if err != nil {
			t.Fatalf("seek(%d) failed: %v", c.target, err)
		}
		if idx != c.idx || acc != c.acc {
			t.Fatalf("seek(%d): got (idx=%d, acc=%d), want (idx=%d, acc=%d)",
				c.target, idx, acc, c.idx, c.acc)
		}
	}
}

func TestCursorSeekItemBytes(t *testing.T) {
	tree, err := New(Config[textChunk, textSummary, NO_EXT]{
		Monoid: textMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), fromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	cursor, err := NewCursor(tree, byteDimension{})
	if err != nil {
		t.Fatalf("new cursor failed: %v", err)
	}

	type tc struct {
		target uint64
		idx    int
		item   string
		acc    uint64
		found  bool
	}
	cases := []tc{
		{target: 0, idx: 0, item: "", acc: 0, found: false},
		{target: 1, idx: 0, item: "ab", acc: 2, found: true},
		{target: 2, idx: 0, item: "ab", acc: 2, found: true},
		{target: 3, idx: 1, item: "c\n", acc: 4, found: true},
		{target: 4, idx: 1, item: "c\n", acc: 4, found: true},
		{target: 5, idx: 2, item: "de\nf", acc: 8, found: true},
		{target: 9, idx: 3, item: "", acc: 8, found: false},
	}
	for _, c := range cases {
		idx, item, acc, found, err := cursor.SeekItem(c.target)
		if err != nil {
			t.Fatalf("SeekItem(%d) failed: %v", c.target, err)
		}
		if idx != c.idx || acc != c.acc || found != c.found || string(item) != c.item {
			t.Fatalf("SeekItem(%d): got (idx=%d, item=%q, acc=%d, found=%v), want (idx=%d, item=%q, acc=%d, found=%v)",
				c.target, idx, string(item), acc, found, c.idx, c.item, c.acc, c.found)
		}
	}
}

func TestCursorSeekLines(t *testing.T) {
	tree, err := New[textChunk, textSummary](Config[textChunk, textSummary, NO_EXT]{
		Monoid: textMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), fromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	cursor, err := NewCursor[textChunk, textSummary, NO_EXT, uint64](tree, lineDimension{})
	if err != nil {
		t.Fatalf("new cursor failed: %v", err)
	}

	type tc struct {
		target uint64
		idx    int
		acc    uint64
	}
	cases := []tc{
		{target: 0, idx: 0, acc: 0},
		{target: 1, idx: 1, acc: 1},
		{target: 2, idx: 2, acc: 2},
		{target: 3, idx: 3, acc: 2},
	}
	for _, c := range cases {
		idx, acc, err := cursor.Seek(c.target)
		if err != nil {
			t.Fatalf("seek(%d) failed: %v", c.target, err)
		}
		if idx != c.idx || acc != c.acc {
			t.Fatalf("seek(%d): got (idx=%d, acc=%d), want (idx=%d, acc=%d)",
				c.target, idx, acc, c.idx, c.acc)
		}
	}
}

func TestCursorSeekUninitializedFails(t *testing.T) {
	c := &Cursor[textChunk, textSummary, NO_EXT, uint64]{}
	_, _, err := c.Seek(1)
	if err == nil {
		t.Fatalf("expected error for uninitialized cursor")
	}
	_, _, _, _, err = c.SeekItem(1)
	if err == nil {
		t.Fatalf("expected error for uninitialized cursor SeekItem")
	}
}

func TestExtCursorSeekBytes(t *testing.T) {
	tree, err := New[textChunk, textSummary](Config[textChunk, textSummary, uint64]{
		Monoid:    textMonoid{},
		Extension: extBytes{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), fromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	cursor, err := NewExtCursor[textChunk, textSummary, uint64, uint64](tree, Uint64Dimension{})
	if err != nil {
		t.Fatalf("new ext cursor failed: %v", err)
	}

	type tc struct {
		target uint64
		idx    int
		acc    uint64
	}
	cases := []tc{
		{target: 0, idx: 0, acc: 0},
		{target: 1, idx: 0, acc: 2},
		{target: 2, idx: 0, acc: 2},
		{target: 3, idx: 1, acc: 4},
		{target: 4, idx: 1, acc: 4},
		{target: 5, idx: 2, acc: 8},
		{target: 9, idx: 3, acc: 8},
	}
	for _, c := range cases {
		idx, acc, err := cursor.Seek(c.target)
		if err != nil {
			t.Fatalf("seek(%d) failed: %v", c.target, err)
		}
		if idx != c.idx || acc != c.acc {
			t.Fatalf("seek(%d): got (idx=%d, acc=%d), want (idx=%d, acc=%d)",
				c.target, idx, acc, c.idx, c.acc)
		}
	}
}

func TestExtCursorSeekItemBytes(t *testing.T) {
	tree, err := New[textChunk, textSummary](Config[textChunk, textSummary, uint64]{
		Monoid:    textMonoid{},
		Extension: extBytes{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), fromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	cursor, err := NewExtCursor[textChunk, textSummary, uint64, uint64](tree, Uint64Dimension{})
	if err != nil {
		t.Fatalf("new ext cursor failed: %v", err)
	}

	type tc struct {
		target uint64
		idx    int
		item   string
		acc    uint64
		found  bool
	}
	cases := []tc{
		{target: 0, idx: 0, item: "", acc: 0, found: false},
		{target: 1, idx: 0, item: "ab", acc: 2, found: true},
		{target: 2, idx: 0, item: "ab", acc: 2, found: true},
		{target: 3, idx: 1, item: "c\n", acc: 4, found: true},
		{target: 4, idx: 1, item: "c\n", acc: 4, found: true},
		{target: 5, idx: 2, item: "de\nf", acc: 8, found: true},
		{target: 9, idx: 3, item: "", acc: 8, found: false},
	}
	for _, c := range cases {
		idx, item, acc, found, err := cursor.SeekItem(c.target)
		if err != nil {
			t.Fatalf("SeekItem(%d) failed: %v", c.target, err)
		}
		if idx != c.idx || acc != c.acc || found != c.found || string(item) != c.item {
			t.Fatalf("SeekItem(%d): got (idx=%d, item=%q, acc=%d, found=%v), want (idx=%d, item=%q, acc=%d, found=%v)",
				c.target, idx, string(item), acc, found, c.idx, c.item, c.acc, c.found)
		}
	}
}

func TestPrefixExt(t *testing.T) {
	tree, err := New[textChunk, textSummary](Config[textChunk, textSummary, uint64]{
		Monoid:    textMonoid{},
		Extension: extBytes{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), fromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	sum, err := tree.PrefixExt(0)
	if err != nil || sum != 0 {
		t.Fatalf("PrefixExt(0) = (%d, %v), want (0, nil)", sum, err)
	}
	sum, err = tree.PrefixExt(1)
	if err != nil || sum != 2 {
		t.Fatalf("PrefixExt(1) = (%d, %v), want (2, nil)", sum, err)
	}
	sum, err = tree.PrefixExt(2)
	if err != nil || sum != 4 {
		t.Fatalf("PrefixExt(2) = (%d, %v), want (4, nil)", sum, err)
	}
	sum, err = tree.PrefixExt(3)
	if err != nil || sum != 8 {
		t.Fatalf("PrefixExt(3) = (%d, %v), want (8, nil)", sum, err)
	}
	if _, err := tree.PrefixExt(4); err == nil {
		t.Fatalf("expected PrefixExt(4) to fail")
	}
}

func TestExtCursorRequiresConfiguredExtension(t *testing.T) {
	tree, err := New[textChunk, textSummary](Config[textChunk, textSummary, uint64]{
		Monoid: textMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NewExtCursor[textChunk, textSummary, uint64, uint64](tree, Uint64Dimension{}); err == nil {
		t.Fatalf("expected ext cursor creation to fail without configured extension")
	} else if !errors.Is(err, ErrExtensionUnavailable) {
		t.Fatalf("expected ErrExtensionUnavailable, got %v", err)
	}
	if _, err := tree.PrefixExt(0); err == nil {
		t.Fatalf("expected PrefixExt to fail without configured extension")
	} else if !errors.Is(err, ErrExtensionUnavailable) {
		t.Fatalf("expected ErrExtensionUnavailable, got %v", err)
	}
	cursor, err := NewCursor[textChunk, textSummary, uint64, uint64](tree, byteDimension{})
	if err != nil {
		t.Fatalf("unexpected cursor creation error: %v", err)
	}
	if _, _, _, _, err := cursor.SeekItem(1); err != nil {
		t.Fatalf("SeekItem on plain cursor should work without extension, got %v", err)
	}
}
