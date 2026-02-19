package chunk

import (
	"errors"
	"strings"
	"testing"
)

func TestNewBuildsBitmaps(t *testing.T) {
	c, err := New("a\n😀b")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	if c.Len() != 7 {
		t.Fatalf("unexpected len: %d", c.Len())
	}
	// char starts at offsets: 0 ('a'), 1 ('\n'), 2 ('😀'), 6 ('b')
	for _, off := range []int{0, 1, 2, 6} {
		if c.Chars()&bit(off) == 0 {
			t.Fatalf("expected chars bit at %d", off)
		}
	}
	if c.Newlines()&bit(1) == 0 {
		t.Fatalf("expected newline bit at 1")
	}
}

func TestNewRejectsInvalidUTF8(t *testing.T) {
	_, err := New(string([]byte{0xff}))
	if !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("expected ErrInvalidUTF8, got %v", err)
	}
	_, err = NewBytes([]byte{0xff})
	if !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("expected ErrInvalidUTF8 from NewBytes, got %v", err)
	}
}

func TestNewRejectsOversizedText(t *testing.T) {
	_, err := New(strings.Repeat("a", MaxBase+1))
	if !errors.Is(err, ErrChunkTooLarge) {
		t.Fatalf("expected ErrChunkTooLarge, got %v", err)
	}
	_, err = NewBytes([]byte(strings.Repeat("a", MaxBase+1)))
	if !errors.Is(err, ErrChunkTooLarge) {
		t.Fatalf("expected ErrChunkTooLarge from NewBytes, got %v", err)
	}
}

func TestNewBytesCopiesInput(t *testing.T) {
	src := []byte("ab😀\n")
	c, err := NewBytes(src)
	if err != nil {
		t.Fatalf("unexpected NewBytes error: %v", err)
	}
	src[0] = 'X'
	if c.String() != "ab😀\n" {
		t.Fatalf("chunk should not alias source bytes, got %q", c.String())
	}
}

func TestSliceAndSplitAt(t *testing.T) {
	c, err := New("ab😀cd")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	s, err := c.Slice(2, 6)
	if err != nil {
		t.Fatalf("unexpected Slice error: %v", err)
	}
	if s.String() != "😀" {
		t.Fatalf("unexpected slice text: %q", s.String())
	}
	if !s.IsCharBoundary(0) || !s.IsCharBoundary(4) || s.IsCharBoundary(1) {
		t.Fatalf("unexpected slice boundary behavior")
	}
	left, right, err := c.SplitAt(2)
	if err != nil {
		t.Fatalf("unexpected SplitAt error: %v", err)
	}
	if left.String() != "ab" || right.String() != "😀cd" {
		t.Fatalf("unexpected split result: %q | %q", left.String(), right.String())
	}
	_, _, err = c.SplitAt(3)
	if !errors.Is(err, ErrNotCharBoundary) {
		t.Fatalf("expected ErrNotCharBoundary, got %v", err)
	}
}

func TestChunkSliceSliceAndSplitAt(t *testing.T) {
	c, err := New("ab😀cd")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	full := c.AsSlice()
	s, err := full.Slice(2, 8)
	if err != nil {
		t.Fatalf("unexpected ChunkSlice.Slice error: %v", err)
	}
	if s.String() != "😀cd" {
		t.Fatalf("unexpected chunk slice text: %q", s.String())
	}
	left, right, err := s.SplitAt(4)
	if err != nil {
		t.Fatalf("unexpected ChunkSlice.SplitAt error: %v", err)
	}
	if left.String() != "😀" || right.String() != "cd" {
		t.Fatalf("unexpected ChunkSlice split: %q | %q", left.String(), right.String())
	}
}

func TestChunkSliceBoundaryErrors(t *testing.T) {
	c, err := New("ab😀cd")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	s := c.AsSlice()
	_, err = s.Slice(-1, 1)
	if !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds, got %v", err)
	}
	_, err = s.Slice(1, 3)
	if !errors.Is(err, ErrNotCharBoundary) {
		t.Fatalf("expected ErrNotCharBoundary, got %v", err)
	}
	_, _, err = s.SplitAt(3)
	if !errors.Is(err, ErrNotCharBoundary) {
		t.Fatalf("expected ErrNotCharBoundary from split, got %v", err)
	}
}

func TestChunkSliceBytesReturnsCopy(t *testing.T) {
	c, err := New("hello")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	s := c.AsSlice()
	b := s.Bytes()
	b[0] = 'X'
	if s.String() != "hello" {
		t.Fatalf("slice should not alias returned bytes, got %q", s.String())
	}
}

func TestAppendFitAndOverflow(t *testing.T) {
	c1, err := New("abc")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	c2, err := New("😀\n")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	out, ok := c1.Append(c2.AsSlice())
	if !ok {
		t.Fatalf("expected append to fit")
	}
	if out.String() != "abc😀\n" {
		t.Fatalf("unexpected append result: %q", out.String())
	}
	// Original chunk must stay unchanged.
	if c1.String() != "abc" {
		t.Fatalf("original chunk changed: %q", c1.String())
	}

	full, err := New(strings.Repeat("a", MaxBase))
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	one, _ := New("b")
	still, ok := full.Append(one.AsSlice())
	if ok {
		t.Fatalf("expected overflow append to fail")
	}
	if still.String() != full.String() {
		t.Fatalf("overflow append should return unchanged chunk")
	}
}
