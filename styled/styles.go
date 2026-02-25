package styled

import (
	"fmt"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/btree"
)

// --- Styled Text -----------------------------------------------------------

// TextFromString creates a stylable text from a string.
func TextFromString(s string) *Text {
	t := &Text{
		text: cords.FromString(s),
		runs: Runs{},
	}
	return t
}

// TextFromCord creates a stylable text from a cord.
func TextFromCord(text cords.Cord) Text {
	t := Text{
		text: text,
		runs: Runs{},
	}
	return t
}

// Raw returns a copy of the text without any styles.
func (t *Text) Raw() cords.Cord {
	return t.text
}

// StyleAt returns the style at byte position pos of the styled text.
func (t *Text) StyleAt(pos uint64) (Style, uint64, error) {
	if t == nil || pos >= t.Raw().Len() {
		return nil, pos, cords.ErrIndexOutOfBounds
	}
	if t.runs.tree == nil || t.runs.tree.IsEmpty() {
		return nil, pos, cords.ErrIndexOutOfBounds
	}
	r := t.runs
	cursor, err := btree.NewCursor(r.tree, StyleDimension{})
	assert(err == nil, "cursor cannot be nil")
	idx, run, acc, found, err := cursor.SeekItem(pos + 1)
	assert(err == nil, "cursor cannot be unsuccessful")
	if !found || run.length == 0 || acc < run.length {
		return nil, pos, fmt.Errorf("%w: internal inconsistency: dimension broken?",
			cords.ErrIndexOutOfBounds)
	}
	runStart := acc - run.length
	if pos < runStart || pos >= acc {
		return nil, pos, fmt.Errorf("%w: internal inconsistency: dimension broken?",
			cords.ErrIndexOutOfBounds)
	}
	tracer().Debugf("item index: %d", idx)
	return run.style, pos - runStart, nil
}

// EachStyleRun applies a function to each run of a single style.
// pos is the text position of this run of text within the overall
// styled text.
//
// This may be thought of as a “push”-interface to access style runs for a text.
// For a “pull”-interface please refer to interface `itemized.Iterator`.
//
// func (t *Text) EachStyleRun(f func(content string, sty Style, pos uint64) error) error {
// 	t.runs.tree.ForEachItem(func(item Run) bool {
// 		length := item.length
// 		content, err := t.Raw().Report(item.Start(), length)
// 		if err != nil {
// 			return err
// 		}
// 		st := leaf.(*styleLeaf).style
// 		return f(content, st, i)
// 	})
// 	return err
// }

// func (t *Text) RangeStyleRun() iter.Seq2[string, Style] {
// 	return func(yield func(string, Style) bool) {
// 		n := 0
// 		_ = t.EachStyleRun(func(content string, sty Style, pos uint64) (e error) {
// 			if !yield(content, sty) {
// 				return
// 			}
// 			n++
// 			return
// 		})
// 	}
// }

// Style styles a run of text, given the start and end position.
func (t *Text) Style(sty Style, from, to uint64) (*Text, error) {
	var err error
	if t.runs.tree == nil || t.runs.tree.IsEmpty() {
		t.runs, err = initialStyle(t.text.Len(), sty, from, to)
		return t, err
	}
	t.runs, err = t.runs.Style(t.text.Len(), sty, from, to)
	return t, err
}

// Section copies a piece of styled text, delimited by parameters from and to.
// func Section(t *Text, from, to uint64) (*Text, error) {
// 	c, err := cords.Substr(t.Raw(), from, to-from)
// 	if err != nil {
// 		return nil, err
// 	}
// 	section := TextFromCord(c)
// 	if cords.Cord(t.styles()).IsVoid() {
// 		return section, nil
// 	}
// 	s, err := cords.Substr(cords.Cord(t.styles()), from, to-from)
// 	if err != nil {
// 		return nil, err
// 	}
// 	section.runs = runs(s)
// 	return section, nil
// }

// StyleChange holds a style and the text position where the style run starts.
type StyleChange struct {
	Style    Style
	Position uint64
	Length   uint64
}

// StyleRuns returns a slice of style runs for a styled text.
// func (t *Text) StyleRuns() []StyleChange {
// 	return t.styleRuns(0)
// }

// func (t *Text) styleRuns(offset uint64) []StyleChange {
// 	count := cords.Cord(t.runs).FragmentCount()
// 	slice := make([]StyleChange, count)
// 	i := 0
// 	_ = cords.Cord(t.runs).EachLeaf(func(leaf cords.Leaf, pos uint64) error {
// 		style := leaf.(*styleLeaf).style
// 		slice[i].Style = style
// 		slice[i].Position = pos
// 		slice[i].Length = leaf.Weight()
// 		i++
// 		return nil
// 	})
// 	return slice
// }

// --- Runs of Styles --------------------------------------------------------

func merge(runs1, runs2 []Run) []Run {
	if len(runs1) == 0 {
		return runs2
	}
	if len(runs2) == 0 {
		return runs1
	}
	l1 := runs1[len(runs1)-1]
	l2 := runs2[0]
	tracer().Debugf("l1 = %v, l2 = %v", l1, l2)
	if equals(l1.style, l2.style) {
		r := l1
		r.length += l2.length
		rr := append(runs1[:len(runs1)-1], r)
		return append(rr, runs2[1:]...)
	}
	return append(runs1, runs2...)
}

func equals(s1, s2 Style) bool {
	if s1 == nil && s2 == nil {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	return s1.Equals(s2)
}

// Runs hold information about style-formats which have been applied to a text.
// There is no automatic synchronization between the text and the style-formats.
//type runs cords.Cord

// String returns an informational string for these Runs. Clients must not rely
// on the format of the string.
// func (r runs) String() string {
// 	return cords.Cord(r).String()
// }

// Len returns the overall length in bytes for these Runs.
// func (r runs) Len() uint64 {
// 	return (cords.Cord(r)).Len()
// }

// Style represents a styling-format which can be applied to a run of text.
type Style interface {
	Equals(other Style) bool // does this Style look equal or differently than another one ?
	String() string          // return some kind of identifying string
}

// initialStyle applies a style to a range [from,to) of characters. Returns a style set.
// Given range boundaries will silently be restricted to valid text positions without
// flagging an error. This may result in the style not being applied due to an invalid
// range.
func initialStyle(textlen uint64, sty Style, from, to uint64) (Runs, error) {
	runs, err := newRuns()
	if err != nil {
		tracer().Errorf("styled runs: failed to create new runs")
		return runs, err
	}
	spn := toSpan(from, to).contained(textlen)
	if spn.void() || textlen == 0 {
		return runs, nil
	}
	if spn.covers(textlen) {
		tracer().Debugf("new styled runs: spanning whole text")
		// cover the complete text
		run := Run{length: textlen, style: sty}
		if runs.tree, err = runs.tree.InsertAt(0, run); err != nil {
			return runs, err
		}
	} else { // run spans a mid-section of the text
		tracer().Debugf("new styled runs: creating sub-spans: %v", spn)
		if spn.l > 0 {
			run := Run{length: spn.l, style: nil}
			if runs.tree, err = runs.tree.InsertAt(0, run); err != nil {
				return runs, err
			}
		}
		//cb.Append(makeStyleLeaf(sty, spn))
		run := Run{length: spn.len(), style: sty}
		if runs.tree, err = runs.tree.InsertAt(runs.tree.Len(), run); err != nil {
			return runs, err
		}
		if spn.r < textlen {
			//cb.Append(makeStyleLeaf(nil, toSpan(spn.r, text.Len())))
			run := Run{length: textlen - spn.r, style: nil}
			if runs.tree, err = runs.tree.InsertAt(runs.tree.Len(), run); err != nil {
				return runs, err
			}
		}
	}
	return runs, nil
}

// Style adds a style to (possibly already existing) styles for a given range
// and returns the unified style set (in btree format).
func (runs Runs) Style(textlen uint64, sty Style, from, to uint64) (Runs, error) {
	spn := toSpan(from, to).contained(textlen)
	if spn.void() || textlen == 0 {
		return runs, nil
	}
	if runs.tree == nil || runs.tree.IsEmpty() {
		return initialStyle(textlen, sty, from, to)
	}
	cursor, err := btree.NewCursor(runs.tree, StyleDimension{})
	if err != nil {
		return runs, err
	}
	iL, leftRun, leftStart, _, err1 := seekRunForByte(runs.tree, cursor, spn.l)
	iR, rightRun, _, rightEnd, err2 := seekRunForByte(runs.tree, cursor, spn.r-1)
	if err1 != nil || err2 != nil {
		return runs, err1
	}

	repl := make([]Run, 0, 3)
	if spn.l > leftStart {
		repl = append(repl, Run{
			length: spn.l - leftStart,
			style:  leftRun.style,
		})
	}
	repl = append(repl, Run{
		length: spn.len(),
		style:  sty,
	})
	if spn.r < rightEnd {
		repl = append(repl, Run{
			length: rightEnd - spn.r,
			style:  rightRun.style,
		})
	}
	repl = normalizeRunList(repl)
	assert(len(repl) > 0, "impossible: normalized run is void")

	tree, err := runs.tree.DeleteRange(iL, iR-iL+1)
	assert(err == nil, "internal inconsistency")
	tree, err = tree.InsertAt(iL, repl...)
	assert(err == nil, "internal inconsistency")

	leftCanMerge := spn.l == leftStart
	rightCanMerge := spn.r == rightEnd
	insertCount := len(repl)
	leftMerged := false
	if leftCanMerge {
		if tree, leftMerged, err = mergeAdjacentRuns(tree, iL-1); err != nil {
			return runs, err
		}
	}
	if rightCanMerge {
		rightPairLeft := iL + insertCount - 1
		if leftMerged {
			rightPairLeft--
		}
		if tree, _, err = mergeAdjacentRuns(tree, rightPairLeft); err != nil {
			return runs, err
		}
	}
	assert(tree.Summary().length() == textlen, "internal inconsistency")
	return Runs{tree: tree}, nil
}

func normalizeRunList(source []Run) []Run {
	assert(len(source) > 0, "impossible void run list")
	out := make([]Run, 0, len(source))
	for _, run := range source {
		if run.length == 0 {
			continue
		}
		n := len(out)
		if n > 0 && equals(out[n-1].style, run.style) {
			out[n-1].length += run.length
			continue
		}
		out = append(out, run)
	}
	return out
}

func seekRunForByte(
	tree *btree.Tree[Run, Summary, btree.NO_EXT],
	cursor *btree.Cursor[Run, Summary, btree.NO_EXT, uint64],
	pos uint64,
) (index int, run Run, runStart uint64, runEnd uint64, err error) {
	index, runEnd, err = cursor.Seek(pos + 1)
	assert(err == nil, "internal inconsistency")
	assert(index >= 0 && index < tree.Len(), "run lookup index out of range")
	run, err = tree.At(index)
	assert(err == nil, "internal inconsistency")
	runStart = runEnd - run.length
	assert(pos >= runStart && pos < runEnd, "run lookup mismatch")
	return index, run, runStart, runEnd, nil
}

func mergeAdjacentRuns(
	tree *btree.Tree[Run, Summary, btree.NO_EXT],
	left int,
) (*btree.Tree[Run, Summary, btree.NO_EXT], bool, error) {
	assert(tree != nil && left >= 0 && left+1 < tree.Len(), "internal inconsistency")
	a, err := tree.At(left)
	assert(err == nil, "internal inconsistency")
	b, err := tree.At(left + 1)
	assert(err == nil, "internal inconsistency")
	if !equals(a.style, b.style) {
		return tree, false, nil
	}
	merged := Run{length: a.length + b.length, style: a.style}
	tree, err = tree.DeleteRange(left, 2)
	assert(err == nil, "internal inconsistency")
	tree, err = tree.InsertAt(left, merged)
	assert(err == nil, "internal inconsistency")
	return tree, true, nil
}

// --- Styled Leaf -----------------------------------------------------------

type styleLeaf struct {
	style  Style  // applied styles
	length uint64 // length of this style run in bytes
}

// length of the style leaf run in bytes
func (sl styleLeaf) Weight() uint64 {
	return sl.length
}

// produce the leaf fragment as a string; will produce the identifying string of the
// enclosed format.
func (sl styleLeaf) String() string {
	if sl.style == nil {
		return "[no style]"
	}
	return sl.style.String()
}

// substring [i:j], not applicable
func (sl styleLeaf) Substring(uint64, uint64) []byte {
	return []byte(sl.String())
}

// split into 2 leafs at position i, resulting in two equal styles with different
// length < |sl|.
// func (sl styleLeaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
// 	left := &styleLeaf{
// 		style:  sl.style,
// 		length: i,
// 	}
// 	right := &styleLeaf{
// 		style:  sl.style,
// 		length: sl.length - i,
// 	}
// 	return left, right
// }

func makeStyleLeaf(sty Style, spn span) *styleLeaf {
	return &styleLeaf{
		style:  sty,
		length: spn.r - spn.l,
	}
}

//var _ cords.Leaf = &styleLeaf{}

// --- Span ------------------------------------------------------------------

type span struct {
	l uint64
	r uint64
}

func toSpan(from, to uint64) span {
	if from > to {
		from, to = to, from
	}
	return span{from, to}
}

func (spn span) void() bool {
	return spn.r <= spn.l
}

func (spn span) len() uint64 {
	if spn.void() {
		return 0
	}
	return spn.r - spn.l
}

// func (spn span) covers(c cords.Cord) bool {
func (spn span) covers(textlen uint64) bool {
	return spn.l == 0 && spn.r >= textlen
}

// func (spn span) contained(c cords.Cord) span {
func (spn span) contained(textlen uint64) span {
	if spn.r > textlen {
		spn.r = textlen
	}
	return spn
}
