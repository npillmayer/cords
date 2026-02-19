package cords

import (
	"errors"
	"strings"
	"testing"
)

func TestPosFromByteRoundtrip(t *testing.T) {
	c := FromString("a😀b")

	cases := []struct {
		byteOff uint64
		runes   uint64
	}{
		{0, 0},
		{1, 1},
		{5, 2},
		{6, 3},
	}
	for _, tc := range cases {
		p, err := c.PosFromByte(tc.byteOff)
		if err != nil {
			t.Fatalf("PosFromByte(%d) failed: %v", tc.byteOff, err)
		}
		if p.runes != tc.runes {
			t.Fatalf("PosFromByte(%d) runes=%d want=%d", tc.byteOff, p.runes, tc.runes)
		}
		b, err := c.ByteOffset(p)
		if err != nil {
			t.Fatalf("ByteOffset(PosFromByte(%d)) failed: %v", tc.byteOff, err)
		}
		if b != tc.byteOff {
			t.Fatalf("roundtrip byte offset=%d want=%d", b, tc.byteOff)
		}
	}
}

func TestPosFromByteRejectsNonBoundary(t *testing.T) {
	c := FromString("a😀b")
	_, err := c.PosFromByte(2) // inside 4-byte rune
	if !errors.Is(err, ErrIllegalPosition) {
		t.Fatalf("expected ErrIllegalPosition, got %v", err)
	}
}

func TestPosEnd(t *testing.T) {
	c := FromString("a😀b")
	p := c.PosEnd()
	if p.runes != 3 || p.bytepos != 6 {
		t.Fatalf("unexpected PosEnd: %+v", p)
	}
}

func TestByteOffsetRejectsMismatchedPos(t *testing.T) {
	c := FromString("hello")
	_, err := c.ByteOffset(Pos{runes: 1, bytepos: 5})
	if !errors.Is(err, ErrIllegalPosition) {
		t.Fatalf("expected ErrIllegalPosition, got %v", err)
	}
}

func TestByteOffsetDetectsCrossCordPos(t *testing.T) {
	c1 := FromString("a😀b")
	p, err := c1.PosFromByte(5)
	if err != nil {
		t.Fatalf("PosFromByte failed: %v", err)
	}

	c2 := FromString("hello")
	_, err = c2.ByteOffset(p)
	if !errors.Is(err, ErrIllegalPosition) {
		t.Fatalf("expected ErrIllegalPosition, got %v", err)
	}
}

func TestPosFromRunesInternal(t *testing.T) {
	c := FromString("a😀b")
	p, err := c.posFromRunes(2)
	if err != nil {
		t.Fatalf("posFromRunes failed: %v", err)
	}
	if p.bytepos != 5 {
		t.Fatalf("bytepos=%d want=5", p.bytepos)
	}
}

func TestPosConversionAcrossChunkBoundary(t *testing.T) {
	// 63 ASCII runes in first chunk, then a multi-byte rune crossing into the
	// next chunk payload, then one trailing ASCII rune.
	s := strings.Repeat("a", 63) + "😀" + "z"
	c := FromString(s)

	p, err := c.posFromRunes(64) // after 63x 'a' + '😀'
	if err != nil {
		t.Fatalf("posFromRunes failed: %v", err)
	}
	if p.bytepos != 67 {
		t.Fatalf("bytepos=%d want=67", p.bytepos)
	}

	p2, err := c.PosFromByte(67)
	if err != nil {
		t.Fatalf("PosFromByte failed: %v", err)
	}
	if p2.runes != 64 {
		t.Fatalf("runes=%d want=64", p2.runes)
	}
}
