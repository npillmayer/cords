package cords

import (
	"errors"
	"strings"
	"testing"
)

func TestReportRunesBasic(t *testing.T) {
	c := FromString("a😀בc")
	start, err := c.posFromRunes(1)
	if err != nil {
		t.Fatalf("posFromRunes failed: %v", err)
	}
	s, err := c.ReportRunes(start, 2)
	if err != nil {
		t.Fatalf("ReportRunes failed: %v", err)
	}
	if s != "😀ב" {
		t.Fatalf("ReportRunes=%q want=%q", s, "😀ב")
	}
}

func TestSplitRunesBasic(t *testing.T) {
	c := FromString("a😀בc")
	p, err := c.posFromRunes(3)
	if err != nil {
		t.Fatalf("posFromRunes failed: %v", err)
	}
	left, right, err := SplitRunes(c, p)
	if err != nil {
		t.Fatalf("SplitRunes failed: %v", err)
	}
	if left.String() != "a😀ב" || right.String() != "c" {
		t.Fatalf("SplitRunes got %q | %q", left.String(), right.String())
	}
}

func TestReportRunesChunkBoundary(t *testing.T) {
	s := strings.Repeat("a", 63) + "😀" + "z"
	c := FromString(s)
	start, err := c.posFromRunes(63)
	if err != nil {
		t.Fatalf("posFromRunes failed: %v", err)
	}
	out, err := c.ReportRunes(start, 2)
	if err != nil {
		t.Fatalf("ReportRunes failed: %v", err)
	}
	if out != "😀z" {
		t.Fatalf("ReportRunes=%q want=%q", out, "😀z")
	}
}

func TestSplitRunesRejectsForeignPos(t *testing.T) {
	c1 := FromString("a😀b")
	p, err := c1.PosFromByte(5)
	if err != nil {
		t.Fatalf("PosFromByte failed: %v", err)
	}
	c2 := FromString("hello")
	_, _, err = SplitRunes(c2, p)
	if !errors.Is(err, ErrIllegalPosition) {
		t.Fatalf("expected ErrIllegalPosition, got %v", err)
	}
}

func TestReportRunesRejectsOutOfRange(t *testing.T) {
	c := FromString("abc")
	start := c.PosStart()
	_, err := c.ReportRunes(start, 4)
	if !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds, got %v", err)
	}
}
