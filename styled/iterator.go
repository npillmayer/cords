package styled

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"math"

	"github.com/npillmayer/uax/bidi"
	gobidi "golang.org/x/text/unicode/bidi"
)

// StyleChange holds a style and the text position where the style run starts.
type StyleChange struct {
	Style    Style
	Position uint64 // text position in bytes
	Length   uint64 // length of the run in bytes
}

func (bsc StyleChange) String() string {
	return fmt.Sprintf("StCh{St=%v, Pos=%d, Len=%d}",
		bsc.Style, bsc.Position, bsc.Length)
}

// EachStyleRun applies a function to each run of a single style.
// pos is the text position of this run of text within the overall
// styled text.
//
// This may be thought of as a “push”-interface to access style runs for a text.
// For a “pull”-interface please refer to interface `itemized.Iterator`.
func (t Text) EachStyleRun(fn func(content string, sty Style, pos uint64) error) error {
	if fn == nil {
		return ErrIllegalArguments
	} else if t.isUnstyled() {
		return nil
	}
	return t.eachStyleRun(0, func(content string, sty Style, pos, _ uint64) error {
		return fn(content, sty, pos)
	})
}

func (t Text) eachStyleRun(offset uint64,
	fn func(content string, sty Style, pos, length uint64) error) error {
	//
	if fn == nil {
		return ErrIllegalArguments
	} else if t.isUnstyled() {
		return nil
	}
	var pos uint64
	var err error
	t.runs.tree.ForEachItem(func(item Run) bool {
		if err != nil {
			return false
		}
		length := item.length
		content, reportErr := t.Raw().Report(pos, length)
		if reportErr != nil {
			err = reportErr
			return false
		}
		err = fn(content, item.style, offset+pos, length)
		pos += length
		return err == nil
	})
	return err
}

var voidIter = func(yield func(StyleChange, io.Reader) bool) {}

// StyleRanges is an iterator over the style runs of a styled text.
// It returns [StyleChange]s and [io.Reader]s for each run.
func (t Text) StyleRanges() iter.Seq2[StyleChange, io.Reader] {
	if t.IsVoid() {
		return voidIter
	}
	if t.isUnstyled() {
		return func(yield func(StyleChange, io.Reader) bool) {
			rnge := StyleChange{Style: nil, Position: 0, Length: t.text.Len()}
			yield(rnge, t.text.Reader())
		}
	}
	var offset uint64
	return func(yield func(StyleChange, io.Reader) bool) {
		t.runs.tree.ForEachItem(func(run Run) bool {
			rnge := StyleChange{
				Style:    run.style,
				Position: offset,
				Length:   run.length,
			}
			reader := t.text.BoundedReader(offset, offset+run.length)
			if !yield(rnge, reader) {
				return false
			}
			offset += run.length
			return true
		})
	}
}

// StyleRuns returns a slice of style runs for a styled text.
// This is a convenience function. If clients want to avoid slice allocation,
// they should use one of the iterators.
func (text *Text) StyleRuns() []StyleChange {
	if text == nil {
		return []StyleChange{}
	}
	var runs []StyleChange
	for run, _ := range text.StyleRanges() {
		runs = append(runs, run)
	}
	return runs
}

// ---Paragraph Iterators ----------------------------------------------------

// StyleRanges is an iterator over the style runs of a styled text.
// It returns [StyleChange]s and [io.Reader]s for each run.
func (para *Paragraph) StyleRanges() iter.Seq2[StyleChange, io.Reader] {
	return para.text.StyleRanges()
}

// BidiStyleChange signals either a style change or a bidi level change.
type BidiStyleChange struct {
	Style    Style
	Position int // text position in bytes
	Length   int // length of the run in bytes
	BidiDir  gobidi.Direction
}

func (bsc BidiStyleChange) String() string {
	return fmt.Sprintf("BidiStCh{st=%v, pos=%d, len=%d, bidi=%v}",
		bsc.Style, bsc.Position, bsc.Length, (bidiDirString(bsc.BidiDir)))
}

func bidiDirString(d gobidi.Direction) string {
	switch d {
	case gobidi.LeftToRight:
		return "LTR"
	case gobidi.RightToLeft:
		return "RTL"
	default:
		return "?"
	}
}

// BidiStyleRanges is an iterator over the style runs of a styled text.
// It intersects the style runs with bidi levels (if present). The iterator
// therefore delivers a new style-run at every boundary of either a style or
// a bidi level.
//
// It returns [BidiStyleChange]s and [io.Reader]s for each run.
func (para *Paragraph) BidiStyleRanges() iter.Seq2[BidiStyleChange, io.Reader] {
	// This is actually a farily complicated logic. We first load the bidi levels,
	// as there will usually be less of them than style runs.
	levels := []bidi.Level{}
	if para.levels == nil {
		levels = append(levels, bidi.Level{From: 0, To: math.MaxInt, BidiClass: gobidi.L})
	} else {
		for _, scrap := range para.levels.Range() {
			tracer().Debugf("level %v", scrap)
			if scrap.To-scrap.From > 0 {
				levels = append(levels, scrap)
			}
		}
	}
	tracer().Debugf("levels = %v", levels)
	levelinx := 0
	level := &levels[levelinx]
	//
	// Now iterate over the style runs. When a style run is present, we will compare
	// iterate over bidi levels until we reach the end of the style run, emitting
	// runs while doing so.
	return func(yield func(BidiStyleChange, io.Reader) bool) {
		for run, reader := range para.text.StyleRanges() {
			assert(reader != nil, "styles: text of style run is nil")
			assert(run.Position+run.Length <= math.MaxInt, "paragraph length overflow")
			runStart := int(run.Position)             // cast safe within a paragraph
			runStop := int(run.Position + run.Length) // cast safe within a paragraph
			//tracer().Debugf("# level = %v, run = %v", level, run)
			var runBytes []byte = nil
			for level.From < runStop {
				nextBndry := min(runStop, level.To)
				styleChange := BidiStyleChange{
					Style:    run.Style,
					Position: level.From,
					Length:   nextBndry - level.From,
					BidiDir:  bidiDirFromClass(level.BidiClass),
				}
				// The iterator contract requires us to yield an io.Reader for the text
				// of the run. We reaceived a reader for the style run from the cord-API.
				// However, if the current bidi-level exceeds the style run, we will need
				// to yield a substring reader.
				var r io.Reader
				if nextBndry == runStop && run.Position == uint64(level.From) {
					r = reader // the bidi level fits into the style run => whole reader
				} else { // otherwise construct a substring reader
					//var err error
					from := level.From - runStart
					to := nextBndry - runStart
					r, runBytes = ensureSubstrReader(runBytes, reader, from, to)
					//tracer().Debugf("runBytes = %v, from=%d, to=%d", runBytes, from, to)
				}
				tracer().Debugf("iterator shipout style change %v", styleChange)
				if !doYield(styleChange, r, yield) { // yield a run (if not of length 0)
					return
				}
				level.From = nextBndry     // advance level position
				if nextBndry == level.To { // if at end of bidi level...
					levelinx++ // ...advance to next bidi level, if any
					if levelinx < len(levels) {
						level = &levels[levelinx]
					}
				}
			}
		}
	}
}

func doYield(b BidiStyleChange, reader io.Reader, yield func(BidiStyleChange, io.Reader) bool) bool {
	// In theory there should be neither 0-length runs of styles or bidi levels.
	assert(b.Length > 0, "styles: 0-length bidi style run")
	assert(reader != nil, "styles: nil reader for bidi style run")
	return yield(b, reader)
}

func ensureSubstrReader(rngBytes []byte, reader io.Reader, from, to int) (io.Reader, []byte) {
	if rngBytes == nil {
		b, err := io.ReadAll(reader)
		// We ensured the reader is not nil. If there is an error, this hints
		// to a problem with the style iterator.
		assert(err == nil, "styles: failed to read text for style run")
		rngBytes = b
	}
	r := bytes.NewReader(rngBytes[from:to])
	return r, rngBytes
}

func bidiDirFromClass(bc gobidi.Class) gobidi.Direction {
	switch bc {
	case gobidi.L:
		return gobidi.LeftToRight
	case gobidi.R:
		return gobidi.RightToLeft
	default:
		return gobidi.LeftToRight
	}
}

// EachStyleRun applies a function to each run of a single style.
// pos is the text position of this run of text within the overall
// styled text, i.e., its included [para.Offset].
//
// This may be thought of as a “push”-interface to access style runs for a text.
// For a “pull”-interface please refer to interface `itemized.Iterator`.
func (text *Paragraph) EachStyleRun(f func(content string, sty Style, pos, length uint64) error) error {
	if text == nil || text.text == nil {
		return nil
	}
	return text.text.eachStyleRun(text.Offset, f)
}

// StyleRuns returns a slice of style runs for a styled paragraph.
func (para *Paragraph) StyleRuns() []StyleChange {
	if para == nil || para.text == nil {
		return nil
	}
	var runs []StyleChange
	for run, _ := range para.text.StyleRanges() {
		run.Position += para.Offset
		runs = append(runs, run)
	}
	return runs
}
