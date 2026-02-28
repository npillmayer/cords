package metrics

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestFindParagraphsLineBreakLF(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.cords")
	defer teardown()
	//
	c := cordext.FromStringNoExt("Alpha\nBeta\nGamma")
	spans := FindParagraphs(c, ParagraphPolicy{})
	want := []ParagraphSpan{
		{From: 0, To: 5},
		{From: 6, To: 10},
		{From: 11, To: 16},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected spans: got=%+v want=%+v", spans, want)
	}
}

func TestFindParagraphsLineBreakCRLF(t *testing.T) {
	c := cordext.FromStringNoExt("A\r\nB\r\nC")
	spans := FindParagraphs(c, ParagraphPolicy{})
	want := []ParagraphSpan{
		{From: 0, To: 1},
		{From: 3, To: 4},
		{From: 6, To: 7},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected spans: got=%+v want=%+v", spans, want)
	}
}

func TestFindParagraphsCRLFAcrossChunkBoundary(t *testing.T) {
	prefix := strings.Repeat("x", chunk.MaxBase-1)
	s := prefix + "\r\nZ"
	c := cordext.FromStringNoExt(s)
	spans := FindParagraphs(c, ParagraphPolicy{})
	want := []ParagraphSpan{
		{From: 0, To: uint64(len(prefix))},
		{From: uint64(len(prefix)) + 2, To: uint64(len(prefix)) + 3},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected spans: got=%+v want=%+v", spans, want)
	}
}

func TestFindParagraphsBlankLines(t *testing.T) {
	c := cordext.FromStringNoExt("a\n\nb\n\nc")
	spans := FindParagraphs(c, ParagraphPolicy{Delimiters: ParagraphByBlankLines})
	want := []ParagraphSpan{
		{From: 0, To: 1},
		{From: 3, To: 4},
		{From: 6, To: 7},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected spans: got=%+v want=%+v", spans, want)
	}
}

func TestFindParagraphsKeepEmpty(t *testing.T) {
	c := cordext.FromStringNoExt("a\n\nb\n")
	spans := FindParagraphs(c, ParagraphPolicy{KeepEmpty: true})
	want := []ParagraphSpan{
		{From: 0, To: 1},
		{From: 2, To: 2},
		{From: 3, To: 4},
		{From: 5, To: 5},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected spans: got=%+v want=%+v", spans, want)
	}
}

func TestFindParagraphsStandaloneCRPolicy(t *testing.T) {
	c := cordext.FromStringNoExt("a\rb\rc")
	spans := FindParagraphs(c, ParagraphPolicy{})
	if len(spans) != 1 || spans[0] != (ParagraphSpan{From: 0, To: 5}) {
		t.Fatalf("default policy should not split on standalone CR: got=%+v", spans)
	}
	spans = FindParagraphs(c, ParagraphPolicy{TreatCRAsLineBreak: true})
	want := []ParagraphSpan{
		{From: 0, To: 1},
		{From: 2, To: 3},
		{From: 4, To: 5},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected spans with CR policy: got=%+v want=%+v", spans, want)
	}
}

func TestFindParagraphsVoidCord(t *testing.T) {
	c2 := cordext.FromStringNoExt("")
	if spans := FindParagraphs(c2, ParagraphPolicy{}); spans != nil {
		t.Fatalf("expected nil spans for void text, got=%+v", spans)
	}
}

func TestParagraphAt(t *testing.T) {
	c := cordext.FromStringNoExt("Alpha\nBeta\nGamma")
	sp, err := ParagraphAt(c, 1, ParagraphPolicy{})
	if err != nil {
		t.Fatalf("ParagraphAt(1) failed: %v", err)
	}
	if sp != (ParagraphSpan{From: 0, To: 5}) {
		t.Fatalf("unexpected span at pos=1: got=%+v", sp)
	}
	sp, err = ParagraphAt(c, 7, ParagraphPolicy{})
	if err != nil {
		t.Fatalf("ParagraphAt(7) failed: %v", err)
	}
	if sp != (ParagraphSpan{From: 6, To: 10}) {
		t.Fatalf("unexpected span at pos=7: got=%+v", sp)
	}
	if _, err := ParagraphAt(c, 5, ParagraphPolicy{}); !errors.Is(err, cordext.ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for separator pos, got %v", err)
	}
	if _, err := ParagraphAt(c, c.Len(), ParagraphPolicy{}); !errors.Is(err, cordext.ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for out-of-bounds pos, got %v", err)
	}
}

func TestParagraphsInRange(t *testing.T) {
	c := cordext.FromStringNoExt("A\nB\n\nC")
	spans, err := ParagraphsInRange(c, 2, 6, ParagraphPolicy{})
	if err != nil {
		t.Fatalf("ParagraphsInRange failed: %v", err)
	}
	want := []ParagraphSpan{
		{From: 2, To: 3},
		{From: 5, To: 6},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected overlapping spans: got=%+v want=%+v", spans, want)
	}

	spans, err = ParagraphsInRange(c, 0, 5, ParagraphPolicy{Delimiters: ParagraphByBlankLines})
	if err != nil {
		t.Fatalf("ParagraphsInRange(blank-lines) failed: %v", err)
	}
	want = []ParagraphSpan{
		{From: 0, To: 3},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected blank-line spans: got=%+v want=%+v", spans, want)
	}
}

func TestParagraphsInRangeBoundsValidation(t *testing.T) {
	c := cordext.FromStringNoExt("abc")
	if _, err := ParagraphsInRange(c, 2, 1, ParagraphPolicy{}); !errors.Is(err, cordext.ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for from>to, got %v", err)
	}
	if _, err := ParagraphsInRange(c, 0, 4, ParagraphPolicy{}); !errors.Is(err, cordext.ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for to>len, got %v", err)
	}
	spans, err := ParagraphsInRange(c, 1, 1, ParagraphPolicy{})
	if err != nil {
		t.Fatalf("unexpected error for empty range: %v", err)
	}
	if spans != nil {
		t.Fatalf("expected nil spans for empty range, got=%+v", spans)
	}

	empty := cordext.FromStringNoExt("")
	spans, err = ParagraphsInRange(empty, 0, 0, ParagraphPolicy{})
	if err != nil {
		t.Fatalf("unexpected error for empty text empty-range: %v", err)
	}
	if spans != nil {
		t.Fatalf("expected nil spans for empty text, got=%+v", spans)
	}
}
