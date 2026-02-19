package btree

import (
	"testing"

	"github.com/npillmayer/cords/chunk"
)

func mustChunk(t *testing.T, s string) chunk.Chunk {
	t.Helper()
	c, err := chunk.New(s)
	if err != nil {
		t.Fatalf("chunk.New(%q) failed: %v", s, err)
	}
	return c
}

func TestTreeWithChunkItemsAndSummaryDimensions(t *testing.T) {
	tree, err := New[chunk.Chunk, chunk.Summary](Config[chunk.Summary]{
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

	byteCur, err := NewCursor[chunk.Chunk, chunk.Summary, uint64](tree, chunk.ByteDimension{})
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

	charCur, err := NewCursor[chunk.Chunk, chunk.Summary, uint64](tree, chunk.CharDimension{})
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

	lineCur, err := NewCursor[chunk.Chunk, chunk.Summary, uint64](tree, chunk.LineDimension{})
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
