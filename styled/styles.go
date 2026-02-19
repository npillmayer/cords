package styled

import (
	"iter"

	"github.com/npillmayer/cords"
)

// --- Styled Text -----------------------------------------------------------

// Text is a styled text. Its text and its styles are automatically synchronized.
type Text struct {
	text cords.Cord
	runs runs
}

// TextFromString creates a stylable text from a string.
func TextFromString(s string) *Text {
	t := &Text{
		text: cords.FromString(s),
		runs: runs{},
	}
	return t
}

// TextFromCord creates a stylable text from a cord.
func TextFromCord(text cords.Cord) *Text {
	t := &Text{
		text: text,
		runs: runs{},
	}
	return t
}

// Raw returns a copy of the text without any styles.
func (t *Text) Raw() cords.Cord {
	return t.text
}

// Styles returns a copy of the text's style runs.
func (t *Text) styles() runs {
	return t.runs
}

// StyleAt returns the style at byte position pos of the styled text.
func (t *Text) StyleAt(pos uint64) (Style, uint64, error) {
	r := cords.Cord(t.runs)
	if r.IsVoid() {
		return nil, pos, cords.ErrIndexOutOfBounds
	}
	leaf, i, err := r.Index(pos)
	if err != nil {
		return nil, pos, err
	}
	if l, ok := leaf.(styleLeaf); ok {
		return l.style, pos, nil
	}
	return nil, i, cords.ErrIllegalArguments
}

// EachStyleRun applies a function to each run of a single style.
// pos is the text position of this run of text within the overall
// styled text.
//
// This may be thought of as a “push”-interface to access style runs for a text.
// For a “pull”-interface please refer to interface `itemized.Iterator`.
func (t *Text) EachStyleRun(f func(content string, sty Style, pos uint64) error) error {
	err := cords.Cord(t.styles()).EachLeaf(func(leaf cords.Leaf, i uint64) error {
		length := leaf.Weight()
		content, err := t.Raw().Report(i, length)
		if err != nil {
			return err
		}
		st := leaf.(*styleLeaf).style
		return f(content, st, i)
	})
	return err
}

func (t *Text) RangeStyleRun() iter.Seq2[string, Style] {
	return func(yield func(string, Style) bool) {
		n := 0
		_ = t.EachStyleRun(func(content string, sty Style, pos uint64) (e error) {
			if !yield(content, sty) {
				return
			}
			n++
			return
		})
	}
}

// Style styles a run of text, given the start and end position.
func (t *Text) Style(sty Style, from, to uint64) *Text {
	if cords.Cord(t.runs).IsVoid() {
		t.runs = applyStyle(t.text, sty, from, to)
		return t
	}
	t.runs = t.runs.Style(sty, from, to)
	return t
}

// Section copies a piece of styled text, delimited by parameters from and to.
func Section(t *Text, from, to uint64) (*Text, error) {
	c, err := cords.Substr(t.Raw(), from, to-from)
	if err != nil {
		return nil, err
	}
	section := TextFromCord(c)
	if cords.Cord(t.styles()).IsVoid() {
		return section, nil
	}
	s, err := cords.Substr(cords.Cord(t.styles()), from, to-from)
	if err != nil {
		return nil, err
	}
	section.runs = runs(s)
	return section, nil
}

// StyleChange holds a style and the text position where the style run starts.
type StyleChange struct {
	Style    Style
	Position uint64
	Length   uint64
}

// StyleRuns returns a slice of style runs for a styled text.
func (t *Text) StyleRuns() []StyleChange {
	return t.styleRuns(0)
}

func (t *Text) styleRuns(offset uint64) []StyleChange {
	count := cords.Cord(t.runs).FragmentCount()
	slice := make([]StyleChange, count)
	i := 0
	_ = cords.Cord(t.runs).EachLeaf(func(leaf cords.Leaf, pos uint64) error {
		style := leaf.(*styleLeaf).style
		slice[i].Style = style
		slice[i].Position = pos
		slice[i].Length = leaf.Weight()
		i++
		return nil
	})
	return slice
}

// --- Runs of Styles --------------------------------------------------------

// Runs hold information about style-formats which have been applied to a text.
// There is no automatic synchronization between the text and the style-formats.
type runs cords.Cord

// String returns an informational string for these Runs. Clients must not rely
// on the format of the string.
func (r runs) String() string {
	return cords.Cord(r).String()
}

// Len returns the overall length in bytes for these Runs.
func (r runs) Len() uint64 {
	return (cords.Cord(r)).Len()
}

// Style represents a styling-format which can be applied to a run of text.
type Style interface {
	Equals(other Style) bool // does this Style look equal or differently than another one ?
	String() string          // return some kind of identifying string
}

// applyStyle applies a style to a range [from,to) of characters. Returns a style set.
// Given range boundaries will silently be restricted to valid text positions without
// flagging an error. This may result in the style not being applied due to an invalid
// range.
func applyStyle(text cords.Cord, sty Style, from, to uint64) runs {
	spn := toSpan(from, to).contained(text)
	cb := cords.NewBuilder()
	if spn.void() || spn.covers(text) {
		cb.Append(makeStyleLeaf(sty, spn))
	} else { // run spans a mid-section of the text
		if spn.l > 0 {
			cb.Append(makeStyleLeaf(nil, toSpan(0, spn.l)))
		}
		cb.Append(makeStyleLeaf(sty, spn))
		if spn.r < text.Len() {
			cb.Append(makeStyleLeaf(nil, toSpan(spn.r, text.Len())))
		}
	}
	return runs(cb.Cord())
}

// Style adds a style to already existing styles and returns the unified set.
func (r runs) Style(sty Style, from, to uint64) runs {
	if cords.Cord(r).IsVoid() {
		tracer().Errorf("styled runs: runs are void, cannot style")
		return r
	}
	spn := toSpan(from, to).contained(cords.Cord(r))
	if spn.void() {
		tracer().Errorf("styled runs: illegal span for style, cannot style")
		return r
	}
	tracer().Debugf("====== runs.Style() =========")
	rc, _, err := cords.Cut(cords.Cord(r), spn.l, spn.len())
	if err != nil {
		return r
	}
	tracer().Debugf("r=%s, length=%d", rc, rc.Len())
	cb := cords.NewBuilder()
	cb.Append(makeStyleLeaf(sty, spn))
	newrun := cb.Cord()
	tracer().Debugf("newrun=%s, length=%d", newrun, newrun.Len())
	rc, err = cords.Insert(rc, newrun, spn.l)
	if err != nil {
		tracer().Errorf("styled runs: insert operation returned: %s", err.Error())
	}
	tracer().Debugf("r=%s, length=%d", r, r.Len())
	r = runs(rc)
	return r
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
func (sl styleLeaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	left := &styleLeaf{
		style:  sl.style,
		length: i,
	}
	right := &styleLeaf{
		style:  sl.style,
		length: sl.length - i,
	}
	return left, right
}

func makeStyleLeaf(sty Style, spn span) *styleLeaf {
	return &styleLeaf{
		style:  sty,
		length: spn.r - spn.l,
	}
}

var _ cords.Leaf = &styleLeaf{}

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

func (spn span) covers(c cords.Cord) bool {
	return spn.l == 0 && spn.r >= c.Len()
}

func (spn span) contained(c cords.Cord) span {
	if spn.r > c.Len() {
		spn.r = c.Len()
	}
	return spn
}
