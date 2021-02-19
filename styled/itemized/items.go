package itemized

import "github.com/npillmayer/cords/styled"

type Iterator struct {
	runs    []styled.StyleChange
	inx     int
	lastErr error
}

func IterateText(text *styled.Text) *Iterator {
	iterator := &Iterator{
		runs: text.StyleRuns(),
	}
	return iterator
}

func IterateParagraphText(para *styled.Paragraph) *Iterator {
	iterator := &Iterator{
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

func (it *Iterator) Style() styled.StyleChange {
	if it.inx == 0 || it.lastErr != nil {
		return styled.StyleChange{}
	}
	return it.runs[it.inx-1]
}
