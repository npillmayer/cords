package formatter

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
	"bufio"
	"errors"
	"io"
	"strings"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/itemized"
	"github.com/npillmayer/uax/bidi"
	"github.com/npillmayer/uax/grapheme"
	"github.com/npillmayer/uax/segment"
	"github.com/npillmayer/uax/uax11"
	"github.com/npillmayer/uax/uax14"
)

// Config represents a set of configuration parameters for formatting.
type Config struct {
	LineWidth int            // line width in terms of ‘en’s, i.e. fixed character width
	Justify   bool           // require output lines to be fully justified
	Debug     bool           // output additional information for debugging
	Context   *uax11.Context // language context
}

// Format is an interface for formatting drivers, given an io.Writer
type Format interface {
	Preamble(io.Writer)                         // output a preamble before a styled paragraph
	Postamble(io.Writer)                        // output a postamble
	StyledText(string, styled.Style, io.Writer) // output uniformly styled text run (item)
	LTR(io.Writer)                              // signal the start of a left-to-right run of text
	RTL(io.Writer)                              // signal the start of a right-to-left run of text
	Line(int, int, io.Writer)                   // signal for the start of a new line
	Newline(io.Writer)                          // output an end-of-line delimiter
	NeedsReordering() ReorderFlag               // what kind of re-ordering support does the formatter need?
}

// ReorderFlag is a hint from a Format whether it needs strings handed over reordered in
// some fashion.
type ReorderFlag int

// Different formatters have different capabilities regarding bidirectional text.
const (
	ReorderNone      ReorderFlag = iota // formatter does reordering on its own (e.g., browser)
	ReorderWords                        // formatter will handle RTL words, but not phrases
	ReorderGraphemes                    // formatter relies on application for reordering
)

// Output formats a paragraph of style text using a given formatter.
//
// Neither of the arguments may be nil. However, it is safe to have config.Context
// set to nil. In this case, uax11.LatinContext is used.
//
// TODO do not consume para
func Output(para *styled.Paragraph, out io.Writer, config *Config, format Format) error {
	//
	if para == nil || config == nil || format == nil {
		return errors.New("illegal argument: nil")
	} else if config.Context == nil {
		config.Context = uax11.LatinContext
	}
	breaks := firstFit(para, config.LineWidth, config.Context)
	format.Preamble(out)
	for i, pos := range breaks {
		line, runs, err := para.WrapAt(pos)
		if err != nil {
			T().Errorf("error Paragraph.WrapAt = %v", err)
			return err
		}
		T().Infof("[%3d] \"%s\"", i, line.Raw())
		T().Infof("      with styles = %v", line.StyleRuns())
		T().Infof("      with runs   = %v", runs)
		// iter := itemized.IterateText(line)
		// for iter.Next() {
		// 	text, style, from, to := iter.Style()
		// 	T().Infof("%v: %d…%d = %s", style, from, to, text)
		// }
		for _, run := range runs.Runs {
			if run.Dir == bidi.RightToLeft {
				format.RTL(out) // TODO probably have to reverse graphemes
			} else {
				format.LTR(out)
			}
			segit := run.SegmentIterator(run.IsOpposite(bidi.LeftToRight))
			for segit.Next() {
				dir, from, to := segit.Segment()
				T().Infof("segment (%v): %d…%d", dir, from, to)
				section, err := styled.Section(line, from, to)
				if err != nil {
					return err
				}
				iter := itemized.IterateText(section)
				for iter.Next() {
					text, style, from, to := iter.Style()
					T().Infof("%v: %d…%d = \"%s\"", style, from, to, text)
					if run.IsOpposite(bidi.LeftToRight) {
						text = reorder(text, format.NeedsReordering())
					}
					format.StyledText(text, style, out)
				}
			}
		}
		format.Newline(out)
		T().Infof("----------- 8< ---------------")
	}
	format.Postamble(out)
	return nil
}

// --- Line breaking ---------------------------------------------------------
/*
We do just a simplistics kind of line breaking, using a first fit algorithm.
It does not squash whitespace and simply consideres breaks where UAX#14
recommends.

From Wikipedia:

	1. |  SpaceLeft := LineWidth
	2. |  for each Word in Text
	3. |      if (Width(Word) + SpaceWidth) > SpaceLeft
	4. |           insert line break before Word in Text
	5. |           SpaceLeft := LineWidth - Width(Word)
	6. |      else
	7. |           SpaceLeft := SpaceLeft - (Width(Word) + SpaceWidth)
*/
func firstFit(para *styled.Paragraph, linewidth int, context *uax11.Context) []uint64 {
	//
	linewrap := uax14.NewLineWrap()
	segmenter := segment.NewSegmenter(linewrap)
	spaceleft := linewidth
	segmenter.Init(bufio.NewReader(para.Reader()))
	breaks := make([]uint64, 0, 20)
	prevpos := 0
	linestart := true
	for segmenter.Next() {
		//T().Infof("----------- seg break -------------")
		//p1, _ := segmenter.Penalties()
		frag := string(segmenter.Bytes())
		gstr := grapheme.StringFromString(frag)
		fraglen := uax11.StringWidth(gstr, context)
		//T().Infof("next segment (p=%d): %s   (len=%d|%d)", p1, gstr, fraglen, spaceleft)
		if fraglen >= spaceleft {
			if linestart { // fragment is too long for a line
				pos := prevpos + len(frag)
				breaks = append(breaks, uint64(pos))
				T().Debugf("line break @ %d", prevpos)
				spaceleft = linewidth
			} else { // fragment overshoots line
				breaks = append(breaks, uint64(prevpos))
				T().Debugf("line break @ %d", prevpos)
				spaceleft = linewidth - fraglen
			}
		} else { // no break, just append the fragment to the current line
			spaceleft -= fraglen
			linestart = false
		}
		prevpos += len(frag)
	}
	if spaceleft < linewidth { // we have a partial line to consume
		breaks = append(breaks, para.Raw().Len())
		T().Debugf("line break @ %d", para.Raw().Len())
	}
	return breaks
}

// ---------------------------------------------------------------------------

func reorder(s string, how ReorderFlag) string {
	if how == ReorderNone {
		return s
	}
	if how == ReorderWords {
		T().Errorf("REVERSE WORDS: %s", s)
		seg := segment.NewSegmenter() // uses a simple word breaker
		seg.Init(strings.NewReader(s))
		out := make([]byte, len(s))
		cursor := len(out)
		for seg.Next() {
			word := seg.Bytes()
			cursor -= len(word)
			copy(out[cursor:], word)
		}
		return string(out)
	}
	// fully reorder by graphemes
	gstr := grapheme.StringFromString(s)
	n := gstr.Len()
	out := make([]byte, len(s))
	for i, j := n-1, 0; i >= 0; i-- {
		g := gstr.Nth(i)
		T().Infof("grapheme = '%s' (%d)", g, len(g))
		copy(out[j:], g)
		j += len(g)
	}
	return string(out)
}
