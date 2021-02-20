package itemized

import "github.com/npillmayer/cords/styled"

type Iterator struct {
	runs    []styled.StyleChange
	inx     int
	length  uint64
	lastErr error
}

func IterateText(text *styled.Text) *Iterator {
	iterator := &Iterator{
		runs:   text.StyleRuns(),
		length: text.Raw().Len(),
	}
	return iterator
}

func IterateParagraphText(para *styled.Paragraph) *Iterator {
	iterator := &Iterator{
		runs:   para.StyleRuns(),
		length: para.Raw().Len(),
	}
	return iterator
}

func (it *Iterator) Next() bool {
	if it.lastErr != nil || it.inx >= len(it.runs) {
		return false
	}
	it.inx++
	return true
}

func (it *Iterator) LastError() error {
	return it.lastErr
}

// Style returns the style at the current iterator position, together with
// the text indices [from…to) of the style run.
func (it *Iterator) Style() (styled.Style, uint64, uint64) {
	if it.inx == 0 || it.lastErr != nil {
		return nil, 0, 0
	}
	// end := it.length
	// if it.inx < len(it.runs) {
	// 	end = it.runs[it.inx].Position
	// }
	// return it.runs[it.inx-1].Style, it.runs[it.inx-1].Position, end
	s := it.runs[it.inx-1]
	return s.Style, s.Position, s.Position + s.Length
}
