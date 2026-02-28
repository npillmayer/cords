package styled

import (
	"errors"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/uax/bidi"
)

func TestTextStyleRuns(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdef")
	bold := teststyle("bold")
	var err error
	if text, err = text.Style(bold, 1, 3); err != nil {
		t.Fatalf("style failed: %v", err)
	}
	// TODO
	// got := text.StyleRuns()
	// want := []StyleChange{
	// 	{Style: nil, Position: 0, Length: 1},
	// 	{Style: bold, Position: 1, Length: 2},
	// 	{Style: nil, Position: 3, Length: 3},
	// }
	// if len(got) != len(want) {
	// 	t.Fatalf("StyleRuns len mismatch: got=%d want=%d", len(got), len(want))
	// }
	// for i := range want {
	// 	if got[i].Position != want[i].Position || got[i].Length != want[i].Length || !equals(got[i].Style, want[i].Style) {
	// 		t.Fatalf("StyleRuns[%d] mismatch: got=%+v want=%+v", i, got[i], want[i])
	// 	}
	// }
}

func TestTextEachStyleRun(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdef")
	bold := teststyle("bold")
	var err error
	if text, err = text.Style(bold, 1, 3); err != nil {
		t.Fatalf("style failed: %v", err)
	}

	type seenRun struct {
		content string
		style   Style
		pos     uint64
	}
	seen := make([]seenRun, 0, 3)
	err = text.EachStyleRun(func(content string, sty Style, pos uint64) error {
		seen = append(seen, seenRun{
			content: content,
			style:   sty,
			pos:     pos,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("EachStyleRun failed: %v", err)
	}
	if len(seen) != 3 {
		t.Fatalf("run count mismatch: got=%d want=3", len(seen))
	}
	if seen[0].content != "a" || !equals(seen[0].style, nil) || seen[0].pos != 0 {
		t.Fatalf("run[0] mismatch: %+v", seen[0])
	}
	if seen[1].content != "bc" || !equals(seen[1].style, bold) || seen[1].pos != 1 {
		t.Fatalf("run[1] mismatch: %+v", seen[1])
	}
	if seen[2].content != "def" || !equals(seen[2].style, nil) || seen[2].pos != 3 {
		t.Fatalf("run[2] mismatch: %+v", seen[2])
	}
}

func TestTextEachStyleRunNoRuns(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abc")
	count := 0
	if err := text.EachStyleRun(func(string, Style, uint64) error {
		count++
		return nil
	}); err != nil {
		t.Fatalf("EachStyleRun on unstyled text returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no runs for unstyled text, got=%d", count)
	}
}

func TestTextEachStyleRunNilCallback(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abc")
	if err := text.EachStyleRun(nil); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for nil callback, got %v", err)
	}
}

func TestParagraphStyleRunsAndEachStyleRun(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abCDxy")
	bold := teststyle("bold")
	var err error
	if text, err = text.Style(bold, 2, 4); err != nil {
		t.Fatalf("style failed: %v", err)
	}
	para, err := ParagraphFromText(&text, 1, 5, bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphFromText failed: %v", err)
	}

	gotRuns := para.StyleRuns()
	wantRuns := []StyleChange{
		{Style: nil, Position: 1, Length: 1},
		{Style: bold, Position: 2, Length: 2},
		{Style: nil, Position: 4, Length: 1},
	}
	if len(gotRuns) != len(wantRuns) {
		t.Fatalf("StyleRuns len mismatch: got=%d want=%d", len(gotRuns), len(wantRuns))
	}
	for i := range wantRuns {
		if gotRuns[i].Position != wantRuns[i].Position || gotRuns[i].Length != wantRuns[i].Length || !equals(gotRuns[i].Style, wantRuns[i].Style) {
			t.Fatalf("StyleRuns[%d] mismatch: got=%+v want=%+v", i, gotRuns[i], wantRuns[i])
		}
	}

	type seenRun struct {
		content string
		style   Style
		pos     uint64
		length  uint64
	}
	seen := make([]seenRun, 0, 3)
	err = para.EachStyleRun(func(content string, sty Style, pos, length uint64) error {
		seen = append(seen, seenRun{
			content: content,
			style:   sty,
			pos:     pos,
			length:  length,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Paragraph.EachStyleRun failed: %v", err)
	}
	if len(seen) != 3 {
		t.Fatalf("run count mismatch: got=%d want=3", len(seen))
	}
	if seen[0].content != "b" || !equals(seen[0].style, nil) || seen[0].pos != 1 || seen[0].length != 1 {
		t.Fatalf("run[0] mismatch: %+v", seen[0])
	}
	if seen[1].content != "CD" || !equals(seen[1].style, bold) || seen[1].pos != 2 || seen[1].length != 2 {
		t.Fatalf("run[1] mismatch: %+v", seen[1])
	}
	if seen[2].content != "x" || !equals(seen[2].style, nil) || seen[2].pos != 4 || seen[2].length != 1 {
		t.Fatalf("run[2] mismatch: %+v", seen[2])
	}
}

func TestParagraphEachStyleRunNilCallback(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abc")
	para, err := ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf("ParagraphFromText failed: %v", err)
	}
	if err := para.EachStyleRun(nil); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for nil callback, got %v", err)
	}
}
