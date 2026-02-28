package styled

import (
	"errors"
	"reflect"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/uax/bidi"
)

func TestParagraphFromTextSection(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	var err error
	text := TextFromString("abCDxy")
	bold := teststyle("bold")
	if text, err = text.Style(bold, 2, 4); err != nil {
		t.Fatalf("style failed: %v", err)
	}
	para, err := ParagraphFromText(&text, 1, 5, bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphFromText failed: %v", err)
	}
	if para.Offset != 1 {
		t.Fatalf("offset mismatch: got=%d want=1", para.Offset)
	}
	if got := para.Raw().String(); got != "bCDx" {
		t.Fatalf("paragraph raw mismatch: got=%q want=%q", got, "bCDx")
	}

	st, off, err := para.StyleAt(0)
	if err != nil {
		t.Fatalf("StyleAt(0) failed: %v", err)
	}
	if !equals(st, nil) || off != 0 {
		t.Fatalf("StyleAt(0) mismatch: style=%v off=%d", st, off)
	}
	st, off, err = para.StyleAt(1)
	if err != nil {
		t.Fatalf("StyleAt(1) failed: %v", err)
	}
	if !equals(st, bold) || off != 0 {
		t.Fatalf("StyleAt(1) mismatch: style=%v off=%d", st, off)
	}
	st, off, err = para.StyleAt(2)
	if err != nil {
		t.Fatalf("StyleAt(2) failed: %v", err)
	}
	if !equals(st, bold) || off != 1 {
		t.Fatalf("StyleAt(2) mismatch: style=%v off=%d", st, off)
	}
}

func TestParagraphFromTextNilInput(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	if _, err := ParagraphFromText(nil, 0, 0, bidi.LeftToRight, nil); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for nil text, got %v", err)
	}
}

func TestFindParagraphSpans(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	text := TextFromString("A\nB\n\nC")
	got := FindParagraphSpans(text, ParagraphPolicy{})
	want := []ParagraphSpan{
		{From: 0, To: 1},
		{From: 2, To: 3},
		{From: 5, To: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("line-break spans mismatch: got=%+v want=%+v", got, want)
	}

	got = FindParagraphSpans(text, ParagraphPolicy{Delimiters: ParagraphByBlankLines})
	want = []ParagraphSpan{
		{From: 0, To: 3},
		{From: 5, To: 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("blank-line spans mismatch: got=%+v want=%+v", got, want)
	}
}

func TestParagraphsFromTextBlankLines(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	var err error
	text := TextFromString("A\n\nB\n\nC")
	bold := teststyle("bold")
	if text, err = text.Style(bold, 3, 4); err != nil {
		t.Fatalf("style failed: %v", err)
	}

	paras, err := ParagraphsFromText(&text, ParagraphPolicy{
		Delimiters: ParagraphByBlankLines,
	}, bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphsFromText failed: %v", err)
	}
	if len(paras) != 3 {
		t.Fatalf("paragraph count mismatch: got=%d want=3", len(paras))
	}
	if paras[0].Offset != 0 || paras[1].Offset != 3 || paras[2].Offset != 6 {
		t.Fatalf("unexpected offsets: got=[%d %d %d]", paras[0].Offset, paras[1].Offset, paras[2].Offset)
	}
	if got := paras[0].Raw().String(); got != "A" {
		t.Fatalf("paragraph[0] mismatch: got=%q want=%q", got, "A")
	}
	if got := paras[1].Raw().String(); got != "B" {
		t.Fatalf("paragraph[1] mismatch: got=%q want=%q", got, "B")
	}
	if got := paras[2].Raw().String(); got != "C" {
		t.Fatalf("paragraph[2] mismatch: got=%q want=%q", got, "C")
	}
	st, off, err := paras[1].StyleAt(0)
	if err != nil {
		t.Fatalf("paragraph[1].StyleAt(0) failed: %v", err)
	}
	if !equals(st, bold) || off != 0 {
		t.Fatalf("paragraph[1].StyleAt(0) mismatch: style=%v off=%d", st, off)
	}
}

func TestParagraphsFromTextOnEmptyText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	text := TextFromString("")
	paras, err := ParagraphsFromText(&text, ParagraphPolicy{}, bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphsFromText(empty) failed: %v", err)
	}
	if len(paras) != 0 {
		t.Fatalf("expected no paragraphs for empty text, got=%d", len(paras))
	}
}

func TestParagraphsFromTextNilInput(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	if _, err := ParagraphsFromText(nil, ParagraphPolicy{}, bidi.LeftToRight, nil); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for nil text, got %v", err)
	}
}

func TestParagraphAtHelper(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	text := TextFromString("A\nB\n\nC")
	sp, err := ParagraphAt(text, 2, ParagraphPolicy{})
	if err != nil {
		t.Fatalf("ParagraphAt failed: %v", err)
	}
	if sp != (ParagraphSpan{From: 2, To: 3}) {
		t.Fatalf("unexpected paragraph span: got=%+v", sp)
	}
	if _, err := ParagraphAt(text, 1, ParagraphPolicy{}); err == nil {
		t.Fatalf("expected error for separator position")
	}
}

func TestParagraphsInRangeHelper(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	text := TextFromString("A\nB\n\nC")
	spans, err := ParagraphsInRange(text, 2, 6, ParagraphPolicy{})
	if err != nil {
		t.Fatalf("ParagraphsInRange failed: %v", err)
	}
	want := []ParagraphSpan{
		{From: 2, To: 3},
		{From: 5, To: 6},
	}
	if !reflect.DeepEqual(spans, want) {
		t.Fatalf("unexpected spans: got=%+v want=%+v", spans, want)
	}
}
