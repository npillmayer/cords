package styled

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	//"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestInitialStyle(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()
	//
	// make a text
	text := cords.FromString("Hello World")
	t.Logf("string='%s', length=%d", text, text.Len())
	// style the text
	bold := teststyle("bold")
	runs, err := applyStyle(text.Len(), bold, 6, text.Len())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("runs.len = %d", runs.tree.Len())
	if runs.tree == nil {
		t.Errorf("expected styles tree to be non-nil")
	} else if runs.tree.Len() != 2 {
		t.Errorf("expected styles tree to have 2 nodes, has %d", runs.tree.Len())
	}
	t.Logf("runs.summary = %v", runs.tree.Summary())
	run, err := runs.tree.At(1)
	if err != nil {
		t.Fatal(err)
	}
	if !equals(bold, run.style) {
		t.Errorf("expected style %v at position 0", bold)
	}
}

func TestBasicStyleCursor(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	text := TextFromString("Hello World, how are you?")
	bold := teststyle("bold")
	runs, err := applyStyle(text.text.Len(), bold, 6, 15)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("runs.len = %d", runs.tree.Len())
	if runs.tree == nil {
		t.Errorf("expected styles tree to be non-nil")
	} else if runs.tree.Len() != 3 {
		t.Errorf("expected styles tree to have 3 nodes, has %d", runs.tree.Len())
	}
	text.runs = runs
	cursor, err := btree.NewCursor(text.runs.tree, StyleDimension{})
	if err != nil {
		t.Fatalf("new cursor failed: %v", err)
	}
	idx, acc, err := cursor.Seek(10)
	if err != nil {
		t.Fatalf("seek(%d) failed: %v", 10, err)
	}
	t.Logf("idx = %d, acc = %v", idx, acc)
	if idx != 1 || acc != 15 {
		t.Fatalf("unexpected seek result: idx=%d acc=%d", idx, acc)
	}
}

func TestBasicStyle(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	text := TextFromString("Hello World, how are you?")
	bold, italic := teststyle("bold"), teststyle("italic")
	text.Style(bold, 6, 11)
	t.Logf("runs.len = %d", text.runs.tree.Len())
	if text.runs.tree == nil {
		t.Errorf("expected styles tree to be non-nil")
	} else if text.runs.tree.Len() != 3 {
		t.Errorf("expected styles tree to have 3 nodes, has %d", text.runs.tree.Len())
	}
	t.Logf("runs.summary = %v", text.runs.tree.Summary())
	text.Style(italic, 8, 16) // erase part of bold run
	summary := text.runs.tree.Summary()
	got := summary.runs
	if len(got) != 4 {
		t.Fatalf("expected 4 style runs, got %d: %+v", len(got), got)
	}
	want := []Run{
		{length: 6, style: nil},
		{length: 2, style: bold},
		{length: 8, style: italic},
		{length: 9, style: nil},
	}
	assertRunsEqual(t, got, want)
	assertRunsInvariant(t, got, text.Raw().Len())
}

func TestStyleMergesAdjacentEqualStyles(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := TextFromString("Hello World, how are you?")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 6, 11); err != nil {
		t.Fatal(err)
	}
	if _, err := text.Style(bold, 11, 16); err != nil {
		t.Fatal(err)
	}
	got := collectRuns(text.runs)
	want := []Run{
		{length: 6, style: nil},
		{length: 10, style: bold},
		{length: 9, style: nil},
	}
	assertRunsEqual(t, got, want)
	assertRunsInvariant(t, got, text.Raw().Len())
}

func TestStyleNoOpOnEmptySpan(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := TextFromString("Hello World")
	if _, err := text.Style(teststyle("bold"), 3, 3); err != nil {
		t.Fatal(err)
	}
	if text.runs.tree != nil && !text.runs.tree.IsEmpty() {
		t.Fatalf("expected empty style tree for empty span, got len=%d", text.runs.tree.Len())
	}
}

func TestStyleCoversWholeText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := TextFromString("Hello World")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 0, text.Raw().Len()); err != nil {
		t.Fatal(err)
	}
	got := collectRuns(text.runs)
	want := []Run{{length: text.Raw().Len(), style: bold}}
	assertRunsEqual(t, got, want)
	assertRunsInvariant(t, got, text.Raw().Len())
}

func TestStyleAtReturnsRunStyleAndRelativeOffset(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := TextFromString("Hello World, how are you?")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 6, 11); err != nil {
		t.Fatal(err)
	}

	type tc struct {
		pos   uint64
		style Style
		off   uint64
	}
	cases := []tc{
		{pos: 0, style: nil, off: 0},
		{pos: 5, style: nil, off: 5},
		{pos: 6, style: bold, off: 0},
		{pos: 8, style: bold, off: 2},
		{pos: 10, style: bold, off: 4},
		{pos: 11, style: nil, off: 0},
	}
	for _, c := range cases {
		gotStyle, gotOff, err := text.StyleAt(c.pos)
		if err != nil {
			t.Fatalf("StyleAt(%d) failed: %v", c.pos, err)
		}
		if !equals(gotStyle, c.style) {
			t.Fatalf("StyleAt(%d) style mismatch: got=%v want=%v", c.pos, gotStyle, c.style)
		}
		if gotOff != c.off {
			t.Fatalf("StyleAt(%d) offset mismatch: got=%d want=%d", c.pos, gotOff, c.off)
		}
	}
}

func TestStyleAtBounds(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := TextFromString("Hello World")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 0, text.Raw().Len()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := text.StyleAt(text.Raw().Len()); !errors.Is(err, cords.ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds at text end, got %v", err)
	}
}

func TestStyleAtWithoutStyleRuns(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	text := TextFromString("abc")
	if _, _, err := text.StyleAt(0); !errors.Is(err, cords.ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds without style runs, got %v", err)
	}
}

// func TestEach(t *testing.T) {
// 	teardown := gotestingadapter.QuickConfig(t, "cords")
// 	defer teardown()
// 	//
// 	text := TextFromString("Hello World, how are you?")
// 	bold := teststyle("bold")
// 	text.Style(bold, 6, 16)
// 	//
// 	cnt := 0
// 	text.EachStyleRun(func(content string, sty Style, pos uint64) error {
// 		cnt++
// 		t.Logf("%v: (%s)", sty, content)
// 		return nil
// 	})
// 	if cnt != 3 {
// 		t.Errorf("expected formatted text to have 3 style runs, has %d", cnt)
// 	}
// }

// func TestRange(t *testing.T) {
// 	teardown := gotestingadapter.QuickConfig(t, "cords")
// 	defer teardown()
// 	//
// 	text := TextFromString("Hello World, how are you?")
// 	normal := teststyle("normal")
// 	bold := teststyle("bold")
// 	text.Style(normal, 0, text.text.Len())
// 	text.Style(bold, 6, 16)
// 	//
// 	cnt := 0
// 	sb := &strings.Builder{}
// 	for run, style := range text.RangeStyleRun() {
// 		cnt++
// 		t.Logf("run: “%s” : %v", run, style)
// 		fmt.Fprintf(sb, "%v", style)
// 	}
// 	if cnt != 3 {
// 		t.Errorf("expected formatted text to have 3 style runs, has %d", cnt)
// 	}
// 	if sb.String() != "[normal][bold][normal]" {
// 		t.Errorf("expected formatted text to have 3 style runs, has %v", sb.String())
// 	}
// }

func TestStyleDetection(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()
	//
}

// --- Test Helpers ----------------------------------------------------------

type mystyle string

func teststyle(sty string) mystyle {
	return mystyle(sty)
}

func (sty mystyle) Equals(other Style) (ok bool) {
	var o mystyle
	if o, ok = other.(mystyle); !ok {
		return false
	}
	return sty == o
}

func (sty mystyle) String() string {
	return fmt.Sprintf("(%s)", string(sty))
}

var _ Style = mystyle("")

func collectRuns(runs Runs) []Run {
	if runs.tree == nil || runs.tree.IsEmpty() {
		return nil
	}
	out := make([]Run, 0, runs.tree.Len())
	runs.tree.ForEachItem(func(item Run) bool {
		out = append(out, item)
		return true
	})
	return out
}

func assertRunsEqual(t *testing.T, got, want []Run) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("run count mismatch: got=%d want=%d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i].length != want[i].length {
			t.Fatalf("run[%d] length mismatch: got=%d want=%d", i, got[i].length, want[i].length)
		}
		if !equals(got[i].style, want[i].style) {
			t.Fatalf("run[%d] style mismatch: got=%v want=%v", i, got[i].style, want[i].style)
		}
	}
}

func assertRunsInvariant(t *testing.T, got []Run, textlen uint64) {
	t.Helper()
	var total uint64
	for i, run := range got {
		if run.length == 0 {
			t.Fatalf("run[%d] has zero length", i)
		}
		total += run.length
		if i > 0 && equals(got[i-1].style, run.style) {
			t.Fatalf("run[%d] and run[%d] should have been merged", i-1, i)
		}
	}
	if total != textlen {
		t.Fatalf("run-length sum mismatch: got=%d want=%d", total, textlen)
	}
}

type testfmtr struct {
	segcnt int
	out    *bytes.Buffer
}

func formatter(prefix string) *testfmtr {
	return &testfmtr{
		out: bytes.NewBufferString(prefix),
	}
}

func (vf testfmtr) String() string {
	return vf.out.String()
}

func (vf *testfmtr) StartRun(f Style, w io.Writer) error {
	vf.segcnt++
	if f == nil {
		_, err := w.Write([]byte("[plain]"))
		return err
	}
	sty := f.(mystyle)
	_, err := w.Write([]byte(sty.String()))
	return err
}

func (vf testfmtr) Format(buf []byte, f Style, w io.Writer) error {
	w.Write(buf)
	return nil
}

func (vf testfmtr) EndRun(f Style, w io.Writer) error {
	_, err := w.Write([]byte("|"))
	return err
}
