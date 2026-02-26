package styled

import (
	"errors"
	"testing"

	"github.com/npillmayer/cords/cordext"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestTextDeleteRangeOnUnstyledText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	var err error
	text := TextFromString("abcdefghij")
	if text, err = text.DeleteRange(3, 6); err != nil {
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

	var err error
	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	if text, err = text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	if text, err = text.DeleteRange(4, 7); err != nil {
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

	var err error
	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	if text, err = text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	if text, err = text.DeleteRange(0, text.Raw().Len()); err != nil {
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

	var err error
	text := TextFromString("abcde")
	if text, err = text.InsertAt(2, cordext.FromStringNoExt("XY"), nil); err != nil {
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
	var err error
	if text, err = text.InsertAt(2, cordext.FromStringNoExt("XY"), bold); err != nil {
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

	var err error
	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	italic := teststyle("italic")
	if text, err = text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}
	if text, err = text.InsertAt(4, cordext.FromStringNoExt("XYZ"), italic); err != nil {
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

	if _, err := text.InsertAt(5, cordext.FromStringNoExt(""), bold); err != nil {
		t.Fatalf("insert no-op failed: %v", err)
	}
	if got := text.Raw().String(); got != origRaw {
		t.Fatalf("no-op changed raw: got=%q want=%q", got, origRaw)
	}
	assertRunsEqual(t, collectRuns(text.runs), origRuns)

	if _, err := text.InsertAt(11, cordext.FromStringNoExt("Z"), bold); !errors.Is(err, ErrIndexOutOfBounds) {
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

	var text Text
	if _, err := text.InsertAt(0, cordext.FromStringNoExt("x"), nil); !errors.Is(err, ErrVoidText) {
		t.Fatalf("expected ErrVoidText for nil text receiver, got %v", err)
	}
}

func TestTextConcatStyledSeamMerge(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	var err error
	left := TextFromString("abcde")
	right := TextFromString("fghij")
	bold := teststyle("bold")
	if left, err = left.Style(bold, 2, 5); err != nil {
		t.Fatal(err)
	}
	if right, err = right.Style(bold, 0, 3); err != nil {
		t.Fatal(err)
	}
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

	var err error
	left := TextFromString("abcde")
	right := TextFromString("XYZ")
	bold := teststyle("bold")
	if left, err = left.Style(bold, 2, 5); err != nil {
		t.Fatal(err)
	}
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
	var err error
	if right, err = right.Style(italic, 1, 4); err != nil {
		t.Fatal(err)
	}
	var l Text
	l, err = left.Concat(right)
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
	var err error
	if left, err = left.Concat(right); err != nil {
		t.Fatalf("concat failed: %v", err)
	}
	if got := left.Raw().String(); got != "abcdef" {
		t.Fatalf("raw text mismatch after concat: got=%q want=%q", got, "abcdef")
	}
	if left.runs.tree != nil && !left.runs.tree.IsEmpty() {
		t.Fatalf("unstyled concat should keep void runs, got len=%d", left.runs.tree.Len())
	}
	if _, _, err = left.StyleAt(0); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds with void runs, got %v", err)
	}
}

func TestTextConcatNilArgument(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	left := TextFromString("abc")
	if _, err := left.Concat(Text{}); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for nil arg, got %v", err)
	}

	var nilText Text
	if _, err := nilText.Concat(left); !errors.Is(err, ErrVoidText) {
		t.Fatalf("expected ErrVoidText for nil receiver, got %v", err)
	}
}

func TestTextSectionOnUnstyledText(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abcdefghij")
	section, err := text.Section(2, 7)
	if err != nil {
		t.Fatalf("section on unstyled text failed: %v", err)
	}
	if got := section.Raw().String(); got != "cdefg" {
		t.Fatalf("raw section mismatch: got=%q want=%q", got, "cdefg")
	}
	if section.runs.tree != nil && !section.runs.tree.IsEmpty() {
		t.Fatalf("unstyled section should keep void runs, got len=%d", section.runs.tree.Len())
	}
}

func TestTextSectionKeepsStyledRunsInSync(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	var err error
	text := TextFromString("abcdefghij")
	bold := teststyle("bold")
	if text, err = text.Style(bold, 2, 8); err != nil {
		t.Fatal(err)
	}

	var section Text
	section, err = text.Section(3, 9)
	if err != nil {
		t.Fatalf("section failed: %v", err)
	}
	if got := section.Raw().String(); got != "defghi" {
		t.Fatalf("raw section mismatch: got=%q want=%q", got, "defghi")
	}
	gotRuns := collectRuns(section.runs)
	wantRuns := []Run{
		{length: 5, style: bold},
		{length: 1, style: nil},
	}
	assertRunsEqual(t, gotRuns, wantRuns)
	assertRunsInvariant(t, gotRuns, section.Raw().Len())

	st, off, err := section.StyleAt(0)
	if err != nil {
		t.Fatalf("StyleAt(0) failed: %v", err)
	}
	if !equals(st, bold) || off != 0 {
		t.Fatalf("StyleAt(0) mismatch: style=%v off=%d", st, off)
	}
	st, off, err = section.StyleAt(5)
	if err != nil {
		t.Fatalf("StyleAt(5) failed: %v", err)
	}
	if !equals(st, nil) || off != 0 {
		t.Fatalf("StyleAt(5) mismatch: style=%v off=%d", st, off)
	}
}

func TestTextSectionBoundsAndNilReceiver(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "styles")
	defer teardown()

	text := TextFromString("abc")
	if _, err := text.Section(2, 1); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for from>to, got %v", err)
	}
	if _, err := text.Section(0, 4); !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds for to>len, got %v", err)
	}

	empty, err := text.Section(1, 1)
	if err != nil {
		t.Fatalf("empty section should succeed: %v", err)
	}
	if empty.Raw().Len() != 0 {
		t.Fatalf("empty section should be length 0, got %d", empty.Raw().Len())
	}

	var nilText Text
	if _, err := nilText.Section(0, 0); !errors.Is(err, ErrVoidText) {
		t.Fatalf("expected ErrVoidText for nil receiver, got %v", err)
	}
}
