package itemized

/*
BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/styled"
)

// Iterator iterates over the style runs (items) of styled text.
type Iterator struct {
	text    cords.Cord
	runs    []styled.StyleChange
	inx     int
	lastErr error
	currinx int
}

// IterateText creates an iterator for styled text.
func IterateText(text *styled.Text) *Iterator {
	iterator := &Iterator{
		text: text.Raw(),
		runs: text.StyleRuns(),
	}
	// If the text has no style runs, create a pseudo-run with nil-style spanning
	// the whole paragraph.
	if len(iterator.runs) == 0 {
		iterator.runs = []styled.StyleChange{{
			Position: 0,
			Length:   text.Raw().Len(),
		}}
	}
	return iterator
}

// IterateParagraphText creates an iterator for styled paragraph of text.
func IterateParagraphText(para *styled.Paragraph) *Iterator {
	iterator := &Iterator{
		text: para.Raw(),
		runs: para.StyleRuns(),
	}
	// If paragraph has no style runs, create a pseudo-run with nil-style spanning
	// the whole paragraph.
	if len(iterator.runs) == 0 {
		iterator.runs = []styled.StyleChange{{
			Position: 0,
			Length:   para.Raw().Len(),
		}}
	}
	return iterator
}

// Next advances the iterator to the next item.
func (it *Iterator) Next() bool {
	if it.lastErr != nil || it.inx >= len(it.runs) {
		return false
	}
	it.inx++
	return true
}

// LastError returns the last error that occured during iteration.
func (it *Iterator) LastError() error {
	return it.lastErr
}

// Style returns the style at the current iterator position, together with
// the text indices [from…to) of the style run.
func (it *Iterator) Style() (string, styled.Style, uint64, uint64) {
	if it.inx == 0 || it.lastErr != nil {
		return "", nil, 0, 0
	}
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
