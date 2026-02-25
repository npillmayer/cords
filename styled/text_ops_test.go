package styled

import (
	"errors"
	"testing"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestTextDeleteRangeOnUnstyledText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdefghij")
	if _, err := text.DeleteRange(3, 6); err != nil {
		t.Fatalf("delete on unstyled text failed: %v", err)
	}
	if got := text.Raw().String(); got != "abcghij" {
		t.Fatalf("raw text mismatch after delete: got=%q want=%q", got, "abcghij")
	}
	if text.runs.tree != nil && !text.runs.tree.IsEmpty() {
		t.Fatalf("unstyled text should keep empty runs, got len=%d", text.runs.tree.Len())
	}
}

func TestTextDeleteRangeKeepsRunsInSync(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := text.DeleteRange(4, 7); err != nil {
		t.Fatalf("delete range failed: %v", err)
	}
	if got := text.Raw().String(); got != "abcdhij" {
		t.Fatalf("raw text mismatch after delete: got=%q want=%q", got, "abcdhij")
	}
	gotRuns := collectRuns(text.runs)
	wantRuns := []Run{
		{length: 2, style: nil},
		{length: 3, style: bold},
		{length: 2, style: nil},
	}
	assertRunsEqual(t, gotRuns, wantRuns)
	assertRunsInvariant(t, gotRuns, text.Raw().Len())

	st, off, err := text.StyleAt(4)
	if err != nil {
		t.Fatalf("StyleAt(4) failed: %v", err)
	}
	if !equals(st, bold) || off != 2 {
		t.Fatalf("StyleAt(4) mismatch: style=%v off=%d", st, off)
	}
	st, off, err = text.StyleAt(5)
	if err != nil {
		t.Fatalf("StyleAt(5) failed: %v", err)
	}
	if !equals(st, nil) || off != 0 {
		t.Fatalf("StyleAt(5) mismatch: style=%v off=%d", st, off)
	}
}

func TestTextDeleteRangeBoundsNoOpAndAtomicity(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	origRaw := text.Raw().String()
	origRuns := collectRuns(text.runs)

	if _, err := text.DeleteRange(5, 5); err != nil {
		t.Fatalf("delete no-op failed: %v", err)
	}
	if got := text.Raw().String(); got != origRaw {
		t.Fatalf("no-op changed raw: got=%q want=%q", got, origRaw)
	}
	assertRunsEqual(t, collectRuns(text.runs), origRuns)

	if _, err := text.DeleteRange(8, 4); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for from>to, got %v", err)
	}
	if got := text.Raw().String(); got != origRaw {
		t.Fatalf("error path changed raw: got=%q want=%q", got, origRaw)
	}
	assertRunsEqual(t, collectRuns(text.runs), origRuns)

	if _, err := text.DeleteRange(0, 11); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for to>len, got %v", err)
	}
	if got := text.Raw().String(); got != origRaw {
		t.Fatalf("error path changed raw: got=%q want=%q", got, origRaw)
	}
	assertRunsEqual(t, collectRuns(text.runs), origRuns)
}

func TestTextDeleteRangeWholeText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := text.DeleteRange(0, text.Raw().Len()); err != nil {
		t.Fatalf("full delete failed: %v", err)
	}
	if text.Raw().Len() != 0 {
		t.Fatalf("expected empty raw text after full delete, got len=%d", text.Raw().Len())
	}
	if got := collectRuns(text.runs); got != nil {
		t.Fatalf("expected empty runs after full delete, got=%+v", got)
	}
	if _, _, err := text.StyleAt(0); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds on empty text, got %v", err)
	}
}

func TestTextInsertAtOnUnstyledText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcde")
	if _, err := text.InsertAt(2, cords.FromString("XY"), nil); err != nil {
		t.Fatalf("insert on unstyled text failed: %v", err)
	}
	if got := text.Raw().String(); got != "abXYcde" {
		t.Fatalf("raw text mismatch after insert: got=%q want=%q", got, "abXYcde")
	}
	if text.runs.tree != nil && !text.runs.tree.IsEmpty() {
		t.Fatalf("unstyled insert with nil style should keep empty runs, got len=%d", text.runs.tree.Len())
	}
}

func TestTextInsertAtOnUnstyledTextWithStyle(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcde")
	bold := teststyle("bold")
	if _, err := text.InsertAt(2, cords.FromString("XY"), bold); err != nil {
		t.Fatalf("styled insert on unstyled text failed: %v", err)
	}
	if got := text.Raw().String(); got != "abXYcde" {
		t.Fatalf("raw text mismatch after insert: got=%q want=%q", got, "abXYcde")
	}
	gotRuns := collectRuns(text.runs)
	wantRuns := []Run{
		{length: 2, style: nil},
		{length: 2, style: bold},
		{length: 3, style: nil},
	}
	assertRunsEqual(t, gotRuns, wantRuns)
	assertRunsInvariant(t, gotRuns, text.Raw().Len())
}

func TestTextInsertAtKeepsRunsInSync(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	italic := teststyle("italic")
	if _, err := text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := text.InsertAt(4, cords.FromString("XYZ"), italic); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if got := text.Raw().String(); got != "abcdXYZefghij" {
		t.Fatalf("raw text mismatch after insert: got=%q want=%q", got, "abcdXYZefghij")
	}
	gotRuns := collectRuns(text.runs)
	wantRuns := []Run{
		{length: 2, style: nil},
		{length: 2, style: bold},
		{length: 3, style: italic},
		{length: 4, style: bold},
		{length: 2, style: nil},
	}
	assertRunsEqual(t, gotRuns, wantRuns)
	assertRunsInvariant(t, gotRuns, text.Raw().Len())

	st, off, err := text.StyleAt(4)
	if err != nil {
		t.Fatalf("StyleAt(4) failed: %v", err)
	}
	if !equals(st, italic) || off != 0 {
		t.Fatalf("StyleAt(4) mismatch: style=%v off=%d", st, off)
	}
	st, off, err = text.StyleAt(8)
	if err != nil {
		t.Fatalf("StyleAt(8) failed: %v", err)
	}
	if !equals(st, bold) || off != 1 {
		t.Fatalf("StyleAt(8) mismatch: style=%v off=%d", st, off)
	}
}

func TestTextInsertAtBoundsNoOpAndAtomicity(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	if _, err := text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	origRaw := text.Raw().String()
	origRuns := collectRuns(text.runs)

	if _, err := text.InsertAt(5, cords.FromString(""), bold); err != nil {
		t.Fatalf("insert no-op failed: %v", err)
	}
	if got := text.Raw().String(); got != origRaw {
		t.Fatalf("no-op changed raw: got=%q want=%q", got, origRaw)
	}
	assertRunsEqual(t, collectRuns(text.runs), origRuns)

	if _, err := text.InsertAt(11, cords.FromString("Z"), bold); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for pos>len, got %v", err)
	}
	if got := text.Raw().String(); got != origRaw {
		t.Fatalf("error path changed raw: got=%q want=%q", got, origRaw)
	}
	assertRunsEqual(t, collectRuns(text.runs), origRuns)
}

func TestTextInsertAtNilReceiver(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	var text *Text
	if _, err := text.InsertAt(0, cords.FromString("x"), nil); !errors.Is(err, ErrVoidText) {
		t.Fatalf("expected ErrVoidText for nil text receiver, got %v", err)
	}
}

func TestTextConcatStyledSeamMerge(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	left := TextFromString("abcde")
	right := TextFromString("fghij")
	bold := teststyle("bold")
	if _, err := left.Style(bold, 2, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := right.Style(bold, 0, 3); err != nil {
		t.Fatal(err)
	}
	var err error
	if left, err = left.Concat(right); err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if got := left.Raw().String(); got != "abcdefghij" {
		t.Fatalf("raw text mismatch after concat: got=%q want=%q", got, "abcdefghij")
	}
	gotRuns := collectRuns(left.runs)
	wantRuns := []Run{
		{length: 2, style: nil},
		{length: 6, style: bold},
		{length: 2, style: nil},
	}
	assertRunsEqual(t, gotRuns, wantRuns)
	assertRunsInvariant(t, gotRuns, left.Raw().Len())
}

func TestTextConcatStyledAndUnstyled(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	left := TextFromString("abcde")
	right := TextFromString("XYZ")
	bold := teststyle("bold")
	if _, err := left.Style(bold, 2, 5); err != nil {
		t.Fatal(err)
	}
	var err error
	if left, err = left.Concat(right); err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if got := left.Raw().String(); got != "abcdeXYZ" {
		t.Fatalf("raw text mismatch after concat: got=%q want=%q", got, "abcdeXYZ")
	}
	gotRuns := collectRuns(left.runs)
	wantRuns := []Run{
		{length: 2, style: nil},
		{length: 3, style: bold},
		{length: 3, style: nil},
	}
	assertRunsEqual(t, gotRuns, wantRuns)
	assertRunsInvariant(t, gotRuns, left.Raw().Len())
}

func TestTextConcatUnstyledAndStyled(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	left := TextFromString("abc")
	right := TextFromString("defg")
	italic := teststyle("italic")
	if _, err := right.Style(italic, 1, 4); err != nil {
		t.Fatal(err)
	}
	l, err := left.Concat(right)
	if err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if l == left {
		t.Fatalf("concat should return a new text, didn't")
	}
	left = l
	if got := left.Raw().String(); got != "abcdefg" {
		t.Fatalf("raw text mismatch after concat: got=%q want=%q", got, "abcdefg")
	}
	gotRuns := collectRuns(left.runs)
	wantRuns := []Run{
		{length: 4, style: nil},
		{length: 3, style: italic},
	}
	assertRunsEqual(t, gotRuns, wantRuns)
	assertRunsInvariant(t, gotRuns, left.Raw().Len())
}

func TestTextConcatBothUnstyledKeepsVoidRuns(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	left := TextFromString("abc")
	right := TextFromString("def")
	if _, err := left.Concat(right); err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if got := left.Raw().String(); got != "abcdef" {
		t.Fatalf("raw text mismatch after concat: got=%q want=%q", got, "abcdef")
	}
	if left.runs.tree != nil && !left.runs.tree.IsEmpty() {
		t.Fatalf("unstyled concat should keep void runs, got len=%d", left.runs.tree.Len())
	}
	if _, _, err := left.StyleAt(0); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds with void runs, got %v", err)
	}
}

func TestTextConcatNilArgument(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	left := TextFromString("abc")
	if _, err := left.Concat(nil); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for nil arg, got %v", err)
	}

	var nilText *Text
	if _, err := nilText.Concat(left); !errors.Is(err, ErrVoidText) {
		t.Fatalf("expected ErrVoidText for nil receiver, got %v", err)
	}
}
