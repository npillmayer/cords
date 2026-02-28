package styled

import (
	"errors"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/uax/bidi"
)

func TestParagraphWrapAtKeepsRunsInSync(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	var err error
	text := TextFromString("abCDxy")
	bold := teststyle("bold")
	if text, err = text.Style(bold, 2, 4); err != nil {
		t.Fatalf("style failed: %v", err)
	}
	para, err := ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphFromText failed: %v", err)
	}

	line, _, err := para.WrapAt(3)
	if err != nil {
		t.Fatalf("WrapAt(3) failed: %v", err)
	}
	if got := line.Raw().String(); got != "abC" {
		t.Fatalf("line raw mismatch: got=%q want=%q", got, "abC")
	}
	if got := para.Raw().String(); got != "Dxy" {
		t.Fatalf("remaining raw mismatch: got=%q want=%q", got, "Dxy")
	}

	// TODO
	// lineRuns := line.StyleRuns()
	// if len(lineRuns) != 2 {
	// 	t.Fatalf("line style runs mismatch: got=%d want=2", len(lineRuns))
	// }
	// if lineRuns[0].Position != 0 || lineRuns[0].Length != 2 || !equals(lineRuns[0].Style, nil) {
	// 	t.Fatalf("line run[0] mismatch: %+v", lineRuns[0])
	// }
	// if lineRuns[1].Position != 2 || lineRuns[1].Length != 1 || !equals(lineRuns[1].Style, bold) {
	// 	t.Fatalf("line run[1] mismatch: %+v", lineRuns[1])
	// }

	st, off, err := para.StyleAt(0)
	if err != nil {
		t.Fatalf("remaining StyleAt(0) failed: %v", err)
	}
	if !equals(st, bold) || off != 0 {
		t.Fatalf("remaining StyleAt(0) mismatch: style=%v off=%d", st, off)
	}
	st, off, err = para.StyleAt(1)
	if err != nil {
		t.Fatalf("remaining StyleAt(1) failed: %v", err)
	}
	if !equals(st, nil) || off != 0 {
		t.Fatalf("remaining StyleAt(1) mismatch: style=%v off=%d", st, off)
	}

	line2, _, err := para.WrapAt(6)
	if err != nil {
		t.Fatalf("WrapAt(6) failed: %v", err)
	}
	if got := line2.Raw().String(); got != "Dxy" {
		t.Fatalf("line2 raw mismatch: got=%q want=%q", got, "Dxy")
	}
	if para.Raw().Len() != 0 {
		t.Fatalf("expected paragraph to be consumed, got len=%d", para.Raw().Len())
	}
	// TODO
	// line2Runs := line2.StyleRuns()
	// if len(line2Runs) != 2 {
	// 	t.Fatalf("line2 style runs mismatch: got=%d want=2", len(line2Runs))
	// }
	// if line2Runs[0].Position != 0 || line2Runs[0].Length != 1 || !equals(line2Runs[0].Style, bold) {
	// 	t.Fatalf("line2 run[0] mismatch: %+v", line2Runs[0])
	// }
	// if line2Runs[1].Position != 1 || line2Runs[1].Length != 2 || !equals(line2Runs[1].Style, nil) {
	// 	t.Fatalf("line2 run[1] mismatch: %+v", line2Runs[1])
	// }
}

func TestParagraphWrapAtUnstyledText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdef")
	para, err := ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphFromText failed: %v", err)
	}
	line, _, err := para.WrapAt(3)
	if err != nil {
		t.Fatalf("WrapAt(3) failed: %v", err)
	}
	if got := line.Raw().String(); got != "abc" {
		t.Fatalf("line raw mismatch: got=%q want=%q", got, "abc")
	}
	if got := para.Raw().String(); got != "def" {
		t.Fatalf("remaining raw mismatch: got=%q want=%q", got, "def")
	}
	// TODO
	// if runs := line.StyleRuns(); runs != nil {
	// 	t.Fatalf("expected nil style runs for unstyled line, got=%+v", runs)
	// }
}

func TestParagraphWrapAtInvalidInputs(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	if _, _, err := (*Paragraph)(nil).WrapAt(0); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for nil paragraph, got %v", err)
	}

	text := TextFromString("abcdef")
	para, err := ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphFromText failed: %v", err)
	}
	if _, _, err := para.WrapAt(3); err != nil {
		t.Fatalf("initial WrapAt failed: %v", err)
	}
	if _, _, err := para.WrapAt(2); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for backward wrap position, got %v", err)
	}
}
