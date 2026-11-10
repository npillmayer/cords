package styled

import (
	"fmt"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/cordext"
)

// --- Styled Text -----------------------------------------------------------

// IsVoid returns true if the text is empty.
// Even if a text is non-empty, it may have no styles.
func (t Text) IsVoid() bool {
	return t.text.IsVoid()
}

// TextFromString creates a stylable text from a string.
func TextFromString(s string) Text {
	t := Text{
		text: cordext.FromStringNoExt(s),
		runs: Runs{},
	}
	return t
}

// TextFromCord creates a stylable text from a cord.
func TextFromCord(text cordext.CordEx[btree.NO_EXT]) Text {
	t := Text{
		text: text,
		runs: Runs{},
	}
	return t
}

// Raw returns a copy of the text without any styles.
func (t *Text) Raw() cordext.CordEx[btree.NO_EXT] {
	return t.text
}

// StyleAt returns the style at byte position pos of the styled text.
func (t *Text) StyleAt(pos uint64) (Style, uint64, error) {
	if t == nil || pos >= t.Raw().Len() {
		return nil, pos, ErrIndexOutOfBounds
	}
	if t.runs.tree == nil || t.runs.tree.IsEmpty() {
		return nil, pos, ErrIndexOutOfBounds
	}
	r := t.runs
	cursor, err := btree.NewCursor(r.tree, styleDimension{})
	assert(err == nil, "cursor cannot be nil")
	idx, run, acc, found, err := cursor.SeekItem(pos + 1)
	assert(err == nil, "cursor cannot be unsuccessful")
	if !found || run.length == 0 || acc < run.length {
		return nil, pos, fmt.Errorf("%w: internal inconsistency: dimension broken?",
			ErrIndexOutOfBounds)
	}
	runStart := acc - run.length
	if pos < runStart || pos >= acc {
		return nil, pos, fmt.Errorf("%w: internal inconsistency: dimension broken?",
			ErrIndexOutOfBounds)
	}
	tracer().Debugf("item index: %d", idx)
	return run.style, pos - runStart, nil
}

// Style applies sty to the half-open UTF-8 byte range [from,to).
// An empty range [p,p), with p <= Raw().Len(), is a no-op returning the
// unchanged text, including its existing style runs, and nil error. This also
// applies to nil styles and empty text. No rune-boundary check is needed for a
// no-op. Reversed ranges, starts beyond the text, and nonempty ranges that
// become empty when the end is clamped return ErrIllegalArguments unchanged.
// For a nonempty range, to is clamped to the text length.
func (t Text) Style(sty Style, from, to uint64) (Text, error) {
	runs, err := t.runs.Style(t.text.Len(), sty, from, to)
	if err != nil {
		return t, err
	}
	t.runs = runs
	return t, nil
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
	//tracer().Debugf("l1 = %v, l2 = %v", l1, l2)
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

// initialStyle creates style runs using the same byte-range contract as Text.Style.
// Valid empty ranges leave the style tree unallocated.
func initialStyle(textlen uint64, sty Style, from, to uint64) (Runs, error) {
	if from == to && from <= textlen {
		return Runs{}, nil
	}
	runs, err := newRuns()
	if err != nil {
		tracer().Errorf("styled runs: failed to create new runs")
		return runs, err
	}
	spn := toSpan(from, to).contained(textlen)
	tracer().Debugf("initial style has span %v", spn)
	if spn.void() {
		return runs, ErrIllegalArguments
	} else if textlen == 0 {
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

// --- Styled Leaf -----------------------------------------------------------

// type styleLeaf struct {
// 	style  Style  // applied styles
// 	length uint64 // length of this style run in bytes
// }

// length of the style leaf run in bytes
// func (sl styleLeaf) Weight() uint64 {
// 	return sl.length
// }

// produce the leaf fragment as a string; will produce the identifying string of the
// enclosed format.
// func (sl styleLeaf) String() string {
// 	if sl.style == nil {
// 		return "[no style]"
// 	}
// 	return sl.style.String()
// }

// substring [i:j], not applicable
// func (sl styleLeaf) Substring(uint64, uint64) []byte {
// 	return []byte(sl.String())
// }

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

// func makeStyleLeaf(sty Style, spn span) *styleLeaf {
// 	return &styleLeaf{
// 		style:  sty,
// 		length: spn.r - spn.l,
// 	}
// }

//var _ cords.Leaf = &styleLeaf{}

// --- Span ------------------------------------------------------------------

type span struct {
	l uint64
	r uint64
}

func toSpan(from, to uint64) span {
	// if from > to {
	// 	from, to = to, from
	// }
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
