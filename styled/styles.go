package styled

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/npillmayer/cords"
)

// --- Runs of Styles --------------------------------------------------------

// Runs hold information about style-formats which have been applied to a text.
// There is not automatic synchronization between the text and the style-formats.
type Runs cords.Cord

// String returns an informational string for these Runs. Clients must not rely
// on the format of the string.
func (runs Runs) String() string {
	return cords.Cord(runs).String()
}

// Len returns the overall length in bytes for these Runs.
func (runs Runs) Len() uint64 {
	return (cords.Cord(runs)).Len()
}

// Format a text. Format reads text from a scanner and applies style-formats to
// runs of text, using a given formatter for output to w.
//
// If any of the arguments is nil, no output is written.
//
func (runs Runs) Format(text *bufio.Scanner, fmtr Formatter, w io.Writer) (err error) {
	if fmtr == nil || text == nil || w == nil {
		return
	}
	remain := uint64(0) // remaining fragment from text.Bytes to format/output
	err = cords.Cord(runs).EachLeaf(func(l cords.Leaf) (leaferr error) {
		style := l.(*styleLeaf)
		if style.Weight() == 0 {
			return nil
		}
		T().Debugf("formatting leaf %v with length=%d", style, style.Weight())
		leaferr = fmtr.StartRun(style.format, w)
		i := uint64(0) // bytes written for this leaf
		for leaferr == nil && i < style.length {
			if remain > 0 { // do not scan new bytes
				T().Debugf("%d bytes remaining to format", remain)
			} else if !text.Scan() {
				T().Errorf("premature end of input text")
				if leaferr = text.Err(); leaferr == nil {
					leaferr = errors.New("premature end of input text")
				} else {
					leaferr = fmt.Errorf("premature end of input text: %w", leaferr)
				}
				break
			} else {
				remain = uint64(len(text.Bytes()))
				T().Debugf("loaded %d new bytes", remain)
			}
			// now remain holds the (suffix) length of text.Bytes not formatted/output yet
			bstart := uint64(len(text.Bytes())) - remain // start within buffer
			l := style.length - i                        // length of substring which may be output
			if l < remain {                              // we output rest of leaf, but not complete buffer
				fmtr.Format(text.Bytes()[bstart:bstart+l], style.format, w)
				remain -= l
				i += l
			} else { // we output a (sub)string of leaf and complete buffer
				fmtr.Format(text.Bytes()[bstart:], style.format, w)
				i += remain
				remain = 0
			}
		}
		if leaferr == nil {
			leaferr = fmtr.EndRun(style.format, w)
		}
		return
	})
	if err != nil && remain > 0 {
		T().Infof("premature end of formatting runs; cannot format rest of input text")
		err = errors.New("premature end of formatting runs; cannot format rest of input text")
	}
	return
}

// Format represents a styling-format which can be applied to a run of text.
type Format interface {
	Equals(Format) bool // does this Format look equal or differently than another one ?
	String() string     // return some kind of identifying string
}

// A Formatter is able to format a run of text according to a style-format.
type Formatter interface {
	StartRun(Format, io.Writer) error
	Format([]byte, Format, io.Writer) error
	EndRun(Format, io.Writer) error
}

// Apply applies a style to a range of characters. Returns a style set.
// Given range boundaries will silently be restricted to valid text positions without
// flagging an error. This may result in the style not being applied due to an invalid
// range.
func Apply(c cords.Cord, sty Format, from, to uint64) Runs {
	spn := toSpan(from, to).contained(c)
	cb := cords.NewBuilder()
	if spn.void() || spn.covers(c) {
		cb.Append(makeStyleLeaf(sty, spn))
	} else { // run spans a mid-section of the text
		if spn.l > 0 {
			cb.Append(makeStyleLeaf(nil, toSpan(0, spn.l)))
		}
		cb.Append(makeStyleLeaf(sty, spn))
		if spn.r < c.Len() {
			cb.Append(makeStyleLeaf(nil, toSpan(spn.r, c.Len())))
		}
	}
	return Runs(cb.Cord())
}

// --- Styled Leaf -----------------------------------------------------------

type styleLeaf struct {
	format Format // applied styles
	length uint64 // length of this style run in bytes
}

// length of the style leaf run in bytes
func (sl styleLeaf) Weight() uint64 {
	return sl.length
}

// produce the leaf fragment as a string; will produce the identifying string of the
// enclosed format.
func (sl styleLeaf) String() string {
	if sl.format == nil {
		return "[no style]"
	}
	return sl.format.String()
}

// substring [i:j], not applicable
func (sl styleLeaf) Substring(uint64, uint64) string {
	return sl.String()
}

// split into 2 leafs at position i, resulting in two equal styles with different
// length < |sl|.
func (sl styleLeaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	left := styleLeaf{
		format: sl.format,
		length: i,
	}
	right := styleLeaf{
		format: sl.format,
		length: sl.length - i,
	}
	return left, right
}

func makeStyleLeaf(sty Format, spn span) *styleLeaf {
	return &styleLeaf{
		format: sty,
		length: spn.r - spn.l,
	}
}

var _ cords.Leaf = styleLeaf{}

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

func (spn span) covers(c cords.Cord) bool {
	return spn.l == 0 && spn.r >= c.Len()
}

func (spn span) contained(c cords.Cord) span {
	if spn.r > c.Len() {
		spn.r = c.Len()
	}
	return spn
}
