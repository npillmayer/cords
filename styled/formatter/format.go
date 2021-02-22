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
	"os"

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
	LineWidth    int
	Justify      bool
	Proportional bool
	Debug        bool
	Context      *uax11.Context
}

// Format is an interface for formatting drivers, given an io.Writer
type Format interface {
	Preamble(io.Writer)
	Postamble(io.Writer)
	StyledText(string, styled.Style, io.Writer)
	LTR(io.Writer)
	RTL(io.Writer)
	Line(int, int, io.Writer)
	Newline(io.Writer)
}

// Output formats a paragraph of style text using a given formatter.
//
// Neither of the arguments may be nil. However, it is safe to have config.Context
// set to nil. In this case, uax11.LatinContext is used.
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
			segit := run.SegmentIterator()
			for segit.Next() {
				dir, from, to := segit.Segment()
				T().Infof("segment (%v): %d…%d", dir, from, to)
				// TODO cut out from…to from line
				section, err := styled.Section(line, from, to)
				if err != nil {
					return err
				}
				iter := itemized.IterateText(section)
				for iter.Next() {
					text, style, from, to := iter.Style()
					T().Infof("%v: %d…%d = \"%s\"", style, from, to, text)
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

// Print outputs a styled paragraph to stdout.
//
// If parameter config is nil,
// a heuristic will create a config from the current terminal's properties (if
// stdout is interactive). Config.Context will also be created based on heuristics
// from the user environment.
func Print(para *styled.Paragraph, config *Config) error {
	if config == nil {
		config = ConfigFromTerminal()
		config.Context = uax11.ContextFromEnvironment()
	}
	consoleFmt := NewConsoleFixedWidthFormat(nil, nil)
	return Output(para, os.Stdout, config, consoleFmt)
}

// --- Line breaking ---------------------------------------------------------
/*
Wikipedia:

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
				T().Infof("break @ %d", prevpos)
				spaceleft = linewidth
			} else { // fragment overshoots line
				breaks = append(breaks, uint64(prevpos))
				T().Infof("break @ %d", prevpos)
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
		T().Infof("break @ %d", para.Raw().Len())
	}
	return breaks
}
