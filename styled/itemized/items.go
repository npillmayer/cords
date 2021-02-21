package itemized

import (
	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/styled"
)

type Iterator struct {
	text    cords.Cord
	runs    []styled.StyleChange
	inx     int
	lastErr error
	currinx int
}

func IterateText(text *styled.Text) *Iterator {
	iterator := &Iterator{
		text: text.Raw(),
		runs: text.StyleRuns(),
	}
	return iterator
}

func IterateParagraphText(para *styled.Paragraph) *Iterator {
	iterator := &Iterator{
		text: para.Raw(),
		runs: para.StyleRuns(),
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
func (it *Iterator) Style() (string, styled.Style, uint64, uint64) {
	if it.inx == 0 || it.lastErr != nil {
		return "", nil, 0, 0
	}
	// end := it.length
	// if it.inx < len(it.runs) {
	// 	end = it.runs[it.inx].Position
	// }
	// return it.runs[it.inx-1].Style, it.runs[it.inx-1].Position, end
	var c cords.Cord
	var err error
	s := it.runs[it.inx-1]
	if it.currinx < it.inx {
		c, err = cords.Substr(it.text, s.Position, s.Length)
		if err != nil {
			T().Errorf("formatter.Style cannot extract text: %s", err.Error())
			panic(err.Error()) // TODO what to do?
		}
		it.currinx = it.inx
	}
	return c.String(), s.Style, s.Position, s.Position + s.Length
}
