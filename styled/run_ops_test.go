package styled

import (
	"errors"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestRunsSplitAtEdges(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	textlen := uint64(20)
	bold := teststyle("bold")
	runs, err := initialStyle(textlen, bold, 6, 11)
	if err != nil {
		t.Fatal(err)
	}

	left, right, err := runs.SplitAt(0)
	if err != nil {
		t.Fatalf("split at 0 failed: %v", err)
	}
	if got := collectRuns(left); got != nil {
		t.Fatalf("left split at 0 should be empty, got=%+v", got)
	}
	assertRunsEqual(t, collectRuns(right), collectRuns(runs))
	assertRunsInvariant(t, collectRuns(right), textlen)

	left, right, err = runs.SplitAt(textlen)
	if err != nil {
		t.Fatalf("split at len failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(left), collectRuns(runs))
	assertRunsInvariant(t, collectRuns(left), textlen)
	if got := collectRuns(right); got != nil {
		t.Fatalf("right split at len should be empty, got=%+v", got)
	}
}

func TestRunsSplitAtRunBoundary(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	textlen := uint64(20)
	bold := teststyle("bold")
	runs, err := initialStyle(textlen, bold, 6, 11)
	if err != nil {
		t.Fatal(err)
	}
	left, right, err := runs.SplitAt(6)
	if err != nil {
		t.Fatalf("split at run boundary failed: %v", err)
	}

	assertRunsEqual(t, collectRuns(left), []Run{
		{length: 6, style: nil},
	})
	assertRunsEqual(t, collectRuns(right), []Run{
		{length: 5, style: bold},
		{length: 9, style: nil},
	})
	assertRunsInvariant(t, collectRuns(left), 6)
	assertRunsInvariant(t, collectRuns(right), 14)
}

func TestRunsSplitAtInsideRun(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	textlen := uint64(20)
	bold := teststyle("bold")
	runs, err := initialStyle(textlen, bold, 6, 11)
	if err != nil {
		t.Fatal(err)
	}
	left, right, err := runs.SplitAt(8)
	if err != nil {
		t.Fatalf("split inside run failed: %v", err)
	}

	assertRunsEqual(t, collectRuns(left), []Run{
		{length: 6, style: nil},
		{length: 2, style: bold},
	})
	assertRunsEqual(t, collectRuns(right), []Run{
		{length: 3, style: bold},
		{length: 9, style: nil},
	})
	assertRunsInvariant(t, collectRuns(left), 8)
	assertRunsInvariant(t, collectRuns(right), 12)
}

func TestRunsSplitAtBounds(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(11, bold, 3, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runs.SplitAt(12); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds, got %v", err)
	}

	empty := Runs{}
	if _, _, err := empty.SplitAt(1); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for empty split, got %v", err)
	}
}

func TestRunsConcatSeamMerge(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	left, err := initialStyle(5, bold, 2, 5) // [2 nil][3 bold]
	if err != nil {
		t.Fatal(err)
	}
	right, err := initialStyle(5, bold, 0, 4) // [4 bold][1 nil]
	if err != nil {
		t.Fatal(err)
	}

	joined, err := left.Concat(right)
	if err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	got := collectRuns(joined)
	want := []Run{
		{length: 2, style: nil},
		{length: 7, style: bold},
		{length: 1, style: nil},
	}
	assertRunsEqual(t, got, want)
	assertRunsInvariant(t, got, 10)
}

func TestRunsConcatNoSeamMerge(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	italic := teststyle("italic")
	left, err := initialStyle(6, bold, 0, 6) // [6 bold]
	if err != nil {
		t.Fatal(err)
	}
	right, err := initialStyle(5, italic, 0, 5) // [5 italic]
	if err != nil {
		t.Fatal(err)
	}

	joined, err := left.Concat(right)
	if err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	got := collectRuns(joined)
	want := []Run{
		{length: 6, style: bold},
		{length: 5, style: italic},
	}
	assertRunsEqual(t, got, want)
	assertRunsInvariant(t, got, 11)
}

func TestRunsConcatWithEmpty(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	nonempty, err := initialStyle(7, bold, 2, 5)
	if err != nil {
		t.Fatal(err)
	}

	var empty Runs
	got, err := empty.Concat(nonempty)
	if err != nil {
		t.Fatalf("empty+nonempty concat failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(got), collectRuns(nonempty))

	got, err = nonempty.Concat(empty)
	if err != nil {
		t.Fatalf("nonempty+empty concat failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(got), collectRuns(nonempty))
}

func TestRunsSectionWholeAndEmpty(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(20, bold, 6, 11)
	if err != nil {
		t.Fatal(err)
	}
	whole, err := runs.Section(0, 20)
	if err != nil {
		t.Fatalf("whole section failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(whole), collectRuns(runs))
	assertRunsInvariant(t, collectRuns(whole), 20)

	empty, err := runs.Section(5, 5)
	if err != nil {
		t.Fatalf("empty section failed: %v", err)
	}
	if got := collectRuns(empty); got != nil {
		t.Fatalf("empty section should have no runs, got=%+v", got)
	}
}

func TestRunsSectionCutsByRange(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(20, bold, 6, 11) // [6 nil][5 bold][9 nil]
	if err != nil {
		t.Fatal(err)
	}

	section, err := runs.Section(8, 14) // [3 bold][3 nil]
	if err != nil {
		t.Fatalf("section failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(section), []Run{
		{length: 3, style: bold},
		{length: 3, style: nil},
	})
	assertRunsInvariant(t, collectRuns(section), 6)

	section, err = runs.Section(7, 10) // inside one run
	if err != nil {
		t.Fatalf("section inside run failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(section), []Run{
		{length: 3, style: bold},
	})
	assertRunsInvariant(t, collectRuns(section), 3)
}

func TestRunsSectionBounds(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(11, bold, 3, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runs.Section(9, 12); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for to>len, got %v", err)
	}
	if _, err := runs.Section(8, 4); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for from>to, got %v", err)
	}

	var empty Runs
	if _, err := empty.Section(0, 0); err != nil {
		t.Fatalf("empty section [0,0) should succeed, got %v", err)
	}
	if _, err := empty.Section(0, 1); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for empty runs section, got %v", err)
	}
}

func TestRunsDeleteRangeNoOpAndBounds(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(11, bold, 3, 9)
	if err != nil {
		t.Fatal(err)
	}
	got, err := runs.DeleteRange(5, 5)
	if err != nil {
		t.Fatalf("delete no-op failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(got), collectRuns(runs))
	assertRunsInvariant(t, collectRuns(got), 11)

	if _, err := runs.DeleteRange(8, 4); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for from>to, got %v", err)
	}
	if _, err := runs.DeleteRange(0, 12); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for to>len, got %v", err)
	}

	var empty Runs
	got, err = empty.DeleteRange(0, 0)
	if err != nil {
		t.Fatalf("empty delete [0,0) should succeed, got %v", err)
	}
	if got.tree != nil && !got.tree.IsEmpty() {
		t.Fatalf("expected empty runs after empty delete, got len=%d", got.tree.Len())
	}
	if _, err := empty.DeleteRange(0, 1); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for empty delete, got %v", err)
	}
}

func TestRunsDeleteRangeCutsAndMerges(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(20, bold, 6, 11) // [6 nil][5 bold][9 nil]
	if err != nil {
		t.Fatal(err)
	}

	after, err := runs.DeleteRange(8, 14) // remove 3 bold + 3 nil
	if err != nil {
		t.Fatalf("delete range failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 6, style: nil},
		{length: 2, style: bold},
		{length: 6, style: nil},
	})
	assertRunsInvariant(t, collectRuns(after), 14)

	after, err = runs.DeleteRange(7, 10) // delete inside one bold run
	if err != nil {
		t.Fatalf("delete within run failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 6, style: nil},
		{length: 2, style: bold},
		{length: 9, style: nil},
	})
	assertRunsInvariant(t, collectRuns(after), 17)
}

func TestRunsDeleteRangeEdgesAndAll(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(20, bold, 6, 11) // [6 nil][5 bold][9 nil]
	if err != nil {
		t.Fatal(err)
	}

	after, err := runs.DeleteRange(0, 6)
	if err != nil {
		t.Fatalf("delete prefix failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 5, style: bold},
		{length: 9, style: nil},
	})
	assertRunsInvariant(t, collectRuns(after), 14)

	after, err = runs.DeleteRange(11, 20)
	if err != nil {
		t.Fatalf("delete suffix failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 6, style: nil},
		{length: 5, style: bold},
	})
	assertRunsInvariant(t, collectRuns(after), 11)

	after, err = runs.DeleteRange(0, 20)
	if err != nil {
		t.Fatalf("delete whole range failed: %v", err)
	}
	if got := collectRuns(after); got != nil {
		t.Fatalf("expected empty runs after full delete, got=%+v", got)
	}
}

func TestRunsInsertAtEmptyAndBounds(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	var empty Runs
	got, err := empty.InsertAt(0, 0, bold)
	if err != nil {
		t.Fatalf("empty insert no-op failed: %v", err)
	}
	if got.tree != nil && !got.tree.IsEmpty() {
		t.Fatalf("expected empty runs for no-op insert, got len=%d", got.tree.Len())
	}

	got, err = empty.InsertAt(0, 4, bold)
	if err != nil {
		t.Fatalf("empty insert failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(got), []Run{{length: 4, style: bold}})
	assertRunsInvariant(t, collectRuns(got), 4)

	if _, err := empty.InsertAt(1, 1, bold); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for empty insert at pos>0, got %v", err)
	}
}

func TestRunsInsertAtNoOpAndBounds(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(11, bold, 3, 9)
	if err != nil {
		t.Fatal(err)
	}
	got, err := runs.InsertAt(5, 0, bold)
	if err != nil {
		t.Fatalf("insert no-op failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(got), collectRuns(runs))
	assertRunsInvariant(t, collectRuns(got), 11)

	if _, err := runs.InsertAt(12, 1, bold); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for pos>len, got %v", err)
	}
}

func TestRunsInsertAtSeamMerges(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	runs, err := initialStyle(20, bold, 6, 11) // [6 nil][5 bold][9 nil]
	if err != nil {
		t.Fatal(err)
	}

	after, err := runs.InsertAt(6, 2, bold) // merge with right bold
	if err != nil {
		t.Fatalf("insert at left seam failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 6, style: nil},
		{length: 7, style: bold},
		{length: 9, style: nil},
	})
	assertRunsInvariant(t, collectRuns(after), 22)

	after, err = runs.InsertAt(11, 2, bold) // merge with left bold
	if err != nil {
		t.Fatalf("insert at right seam failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 6, style: nil},
		{length: 7, style: bold},
		{length: 9, style: nil},
	})
	assertRunsInvariant(t, collectRuns(after), 22)

	after, err = runs.InsertAt(20, 4, nil) // merge with trailing nil
	if err != nil {
		t.Fatalf("insert at end failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 6, style: nil},
		{length: 5, style: bold},
		{length: 13, style: nil},
	})
	assertRunsInvariant(t, collectRuns(after), 24)
}

func TestRunsInsertAtInsideRun(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	bold := teststyle("bold")
	italic := teststyle("italic")
	runs, err := initialStyle(20, bold, 6, 11) // [6 nil][5 bold][9 nil]
	if err != nil {
		t.Fatal(err)
	}
	after, err := runs.InsertAt(8, 3, italic) // split bold run
	if err != nil {
		t.Fatalf("insert inside run failed: %v", err)
	}
	assertRunsEqual(t, collectRuns(after), []Run{
		{length: 6, style: nil},
		{length: 2, style: bold},
		{length: 3, style: italic},
		{length: 3, style: bold},
		{length: 9, style: nil},
	})
	assertRunsInvariant(t, collectRuns(after), 23)
}
