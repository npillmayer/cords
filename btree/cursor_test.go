package btree

import "testing"

func TestCursorSeekBytes(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), FromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	cursor, err := NewCursor[TextChunk, TextSummary, uint64](tree, ByteDimension{})
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

func TestCursorSeekLines(t *testing.T) {
	tree, err := New[TextChunk, TextSummary](Config[TextSummary]{
		Monoid: TextMonoid{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range []string{"ab", "c\n", "de\nf"} {
		tree, err = tree.InsertAt(tree.Len(), FromString(s))
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	cursor, err := NewCursor[TextChunk, TextSummary, uint64](tree, LineDimension{})
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
	c := &Cursor[TextChunk, TextSummary, uint64]{}
	_, _, err := c.Seek(1)
	if err == nil {
		t.Fatalf("expected error for uninitialized cursor")
	}
}
