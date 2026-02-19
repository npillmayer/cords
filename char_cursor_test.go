package cords

import (
	"errors"
	"strings"
	"testing"
)

func TestCharCursorNextPrevRoundtrip(t *testing.T) {
	s := "a😀ב\nz"
	c := FromString(s)
	cc, err := c.NewCharCursor()
	if err != nil {
		t.Fatalf("NewCharCursor failed: %v", err)
	}

	var got []rune
	for {
		r, ok := cc.Next()
		if !ok {
			break
		}
		got = append(got, r)
	}
	want := []rune(s)
	if len(got) != len(want) {
		t.Fatalf("forward rune count=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("forward rune[%d]=%q want=%q", i, got[i], want[i])
		}
	}

	var back []rune
	for {
		r, ok := cc.Prev()
		if !ok {
			break
		}
		back = append(back, r)
	}
	if len(back) != len(want) {
		t.Fatalf("backward rune count=%d want=%d", len(back), len(want))
	}
	for i := range want {
		if back[i] != want[len(want)-1-i] {
			t.Fatalf("backward rune[%d]=%q want=%q", i, back[i], want[len(want)-1-i])
		}
	}
}

func TestCharCursorSeekRunes(t *testing.T) {
	c := FromString("a😀b")
	cc, err := c.NewCharCursor()
	if err != nil {
		t.Fatalf("NewCharCursor failed: %v", err)
	}
	if err := cc.SeekRunes(2); err != nil {
		t.Fatalf("SeekRunes failed: %v", err)
	}
	if cc.ByteOffset() != 5 {
		t.Fatalf("byte offset=%d want=5", cc.ByteOffset())
	}
	r, ok := cc.Next()
	if !ok || r != 'b' {
		t.Fatalf("Next after SeekRunes(2) got (%q,%v), want ('b',true)", r, ok)
	}
}

func TestCharCursorSeekPosRejectsForeignPos(t *testing.T) {
	c1 := FromString("a😀b")
	p, err := c1.PosFromByte(5)
	if err != nil {
		t.Fatalf("PosFromByte failed: %v", err)
	}

	c2 := FromString("hello")
	cc, err := c2.NewCharCursor()
	if err != nil {
		t.Fatalf("NewCharCursor failed: %v", err)
	}
	err = cc.SeekPos(p)
	if !errors.Is(err, ErrIllegalPosition) {
		t.Fatalf("expected ErrIllegalPosition, got %v", err)
	}
}

func TestCharCursorChunkBoundary(t *testing.T) {
	s := strings.Repeat("a", 63) + "😀" + "z"
	c := FromString(s)
	cc, err := c.NewCharCursor()
	if err != nil {
		t.Fatalf("NewCharCursor failed: %v", err)
	}
	if err := cc.SeekRunes(63); err != nil {
		t.Fatalf("SeekRunes(63) failed: %v", err)
	}
	if cc.ByteOffset() != 63 {
		t.Fatalf("byte offset=%d want=63", cc.ByteOffset())
	}

	r, ok := cc.Next()
	if !ok || r != '😀' {
		t.Fatalf("first Next got (%q,%v), want ('😀',true)", r, ok)
	}
	if cc.ByteOffset() != 67 {
		t.Fatalf("byte offset after emoji=%d want=67", cc.ByteOffset())
	}
	r, ok = cc.Next()
	if !ok || r != 'z' {
		t.Fatalf("second Next got (%q,%v), want ('z',true)", r, ok)
	}

	if err := cc.SeekRunes(64); err != nil {
		t.Fatalf("SeekRunes(64) failed: %v", err)
	}
	r, ok = cc.Prev()
	if !ok || r != '😀' {
		t.Fatalf("Prev from rune 64 got (%q,%v), want ('😀',true)", r, ok)
	}
	if cc.ByteOffset() != 63 {
		t.Fatalf("byte offset after Prev=%d want=63", cc.ByteOffset())
	}
}
