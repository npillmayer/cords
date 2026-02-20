package cords

import (
	"errors"
	"strings"
	"testing"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

type newlineExt struct {
	id string
}

func (e newlineExt) MagicID() string {
	if e.id != "" {
		return e.id
	}
	return "cords:test:newlines"
}

func (newlineExt) Zero() uint64 { return 0 }
func (newlineExt) Add(left, right uint64) uint64 {
	return left + right
}

func (newlineExt) FromSegment(seg TextSegment) uint64 {
	return seg.LineCount()
}

type uint64Dim struct{}

func (uint64Dim) Zero() uint64 { return 0 }
func (uint64Dim) Add(acc, s uint64) uint64 {
	return acc + s
}
func (uint64Dim) Compare(acc, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}

func TestCordWithExtensionAggregatesNewlines(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := strings.Repeat("ab\n", 50)
	base := FromString(text)
	cord, err := WithExtension(base, newlineExt{})
	if err != nil {
		t.Fatalf("WithExtension failed: %v", err)
	}
	if cord.String() != text {
		t.Fatalf("string mismatch: got=%q want=%q", cord.String(), text)
	}
	ext, ok := cord.Ext()
	if !ok {
		t.Fatalf("Ext() not available")
	}
	want := uint64(strings.Count(text, "\n"))
	if ext != want {
		t.Fatalf("newline extension mismatch: got=%d want=%d", ext, want)
	}
	if cord.AsCord().String() != text {
		t.Fatalf("AsCord string mismatch: got=%q want=%q", cord.AsCord().String(), text)
	}
}

func TestFromStringWithExtension(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := "Hello\nWorld\n"
	cord, err := FromStringWithExtension(text, newlineExt{})
	if err != nil {
		t.Fatalf("FromStringWithExtension failed: %v", err)
	}
	ext, ok := cord.Ext()
	if !ok {
		t.Fatalf("Ext() not available")
	}
	if ext != 2 {
		t.Fatalf("newline extension mismatch: got=%d want=2", ext)
	}
}

func TestCordExConcatRejectsIncompatibleExtension(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	left, err := FromStringWithExtension("left", newlineExt{id: "cords:test:left"})
	if err != nil {
		t.Fatalf("create left failed: %v", err)
	}
	right, err := FromStringWithExtension("right", newlineExt{id: "cords:test:right"})
	if err != nil {
		t.Fatalf("create right failed: %v", err)
	}
	_, err = left.Concat(right)
	if !errors.Is(err, btree.ErrIncompatibleExtension) {
		t.Fatalf("expected ErrIncompatibleExtension, got %v", err)
	}
}

func TestCordExCursor(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := "a\nb\nc\nd\n"
	cord, err := FromStringWithExtension(text, newlineExt{})
	if err != nil {
		t.Fatalf("FromStringWithExtension failed: %v", err)
	}
	cursor, err := NewExtCursor(cord, uint64Dim{})
	if err != nil {
		t.Fatalf("NewExtCursor failed: %v", err)
	}
	_, acc, err := cursor.Seek(3)
	if err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	if acc < 3 {
		t.Fatalf("seek accumulator too small: got=%d want>=3", acc)
	}
}
