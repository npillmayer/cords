package styled

import (
	"io"
	"iter"
)

// StyleChange holds a style and the text position where the style run starts.
type StyleChange struct {
	Style    Style
	Position uint64 // text position in bytes
	Length   uint64 // length of the run in bytes
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

// StyleRanges is an iterator over the style runs of a styled text.
// It returns [StyleChange]s and [io.Reader]s for each run.
func (t Text) StyleRanges() iter.Seq2[StyleChange, io.Reader] {
	if t.isUnstyled() {
		return nil
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

// ---Paragraph Iterators ----------------------------------------------------

// EachStyleRun applies a function to each run of a single style.
// pos is the text position of this run of text within the overall
// styled text, i.e., its included [para.Offset].
//
// This may be thought of as a “push”-interface to access style runs for a text.
// For a “pull”-interface please refer to interface `itemized.Iterator`.
func (para *Paragraph) EachStyleRun(f func(content string, sty Style, pos, length uint64) error) error {
	if para == nil || para.text == nil {
		return nil
	}
	return para.text.eachStyleRun(para.Offset, f)
}

// StyleRuns returns a slice of style runs for a styled text.
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
