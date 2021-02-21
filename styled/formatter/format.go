package formatter

import (
	"bufio"
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
	for segmenter.Next() {
		//T().Infof("----------- seg break -------------")
		//p1, _ := segmenter.Penalties()
		frag := string(segmenter.Bytes())
		gstr := grapheme.StringFromString(frag)
		fraglen := uax11.StringWidth(gstr, context)
		//T().Infof("next segment (p=%d): %s   (len=%d|%d)", p1, gstr, fraglen, spaceleft)
		if fraglen >= spaceleft { // TODO discard space, if language allows it
			breaks = append(breaks, uint64(prevpos))
			T().Infof("break @ %d", prevpos)
			spaceleft = linewidth - fraglen
		} else {
			spaceleft -= fraglen
			prevpos += len(frag)
		}
	}
	if spaceleft < linewidth {
		breaks = append(breaks, para.Raw().Len())
		T().Infof("break @ %d", para.Raw().Len())
	}
	return breaks
}

type Config struct {
	LineWidth    int
	Justify      bool
	Proportional bool
	Context      *uax11.Context
}

type Format interface {
	Preamble(io.Writer)
	Postamble(io.Writer)
	StyledText(string, styled.Style, io.Writer)
	LTR(io.Writer)
	RTL(io.Writer)
	Line(int, int, io.Writer)
	Newline(io.Writer)
}

func Output(para *styled.Paragraph, out io.Writer, config *Config, format Format) error {
	//
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

func Print(para *styled.Paragraph, config *Config) error {
	if config == nil {
		config = &Config{ // TODO from environment
			LineWidth: 30,
			Context:   uax11.LatinContext,
		}
	}
	consoleFmt := NewConsoleFixedWidthFormat(nil, nil)
	return Output(para, os.Stdout, config, consoleFmt)
}
