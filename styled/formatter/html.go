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
	"io"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/inline"
	"github.com/npillmayer/uax/bidi"
	"github.com/npillmayer/uax/uax11"
)

var htmlStyleNames map[inline.Style]string = map[inline.Style]string{
	inline.PlainStyle:   "",
	inline.BoldStyle:    "b",
	inline.ItalicsStyle: "i",
	inline.StrongStyle:  "strong",
	inline.EmStyle:      "em",
	inline.SmallStyle:   "small",
	inline.MarkedStyle:  "marked",
}

// HTML is a format for simple HTML output.
type HTML struct {
	reorderPolicy ReorderFlag
	dir           bidi.Direction
	bidi          bool
}

// NewHTML creates an HTML formatter.
func NewHTML(reorder ReorderFlag) *HTML {
	html := &HTML{
		reorderPolicy: reorder,
	}
	return html
}

// Print outputs a styled paragraph as HTML.
//
// If parameter config is nil, a default configuration will be used.
// Config.Context will also be created based on heuristics
// from the user environment.
func (html *HTML) Print(para *styled.Paragraph, w io.Writer, config *Config) error {
	if config == nil {
		config = &Config{
			LineWidth: 40,
			Context:   uax11.ContextFromEnvironment(),
		}
	}
	return Output(para, w, config, html)
}

// StyledText is called by the formatting driver to output a sequence of
// uniformly styled text (item).
// (Part of interface Format)
func (html *HTML) StyledText(s string, style styled.Style, w io.Writer) {
	if style == nil {
		w.Write([]byte(s))
		return
	}
	switch st := style.(type) {
	case inline.Style:
		w.Write([]byte(HTMLStyle(st).tags(false)))
		w.Write([]byte(s))
		w.Write([]byte(HTMLStyle(st).tags(true)))
		return
	case HTMLStyle:
		w.Write([]byte(st.tags(false)))
		w.Write([]byte(s))
		w.Write([]byte(st.tags(true)))
		return
	}
	w.Write([]byte(s))
}

// Preamble is called by the output driver before a paragraph of text will be formatted.
// It outputs the a `pre` tag.
// (Part of interface Format)
func (html *HTML) Preamble(w io.Writer) {
	w.Write([]byte("<pre>\n"))
}

// Postamble will be called after a paragraph of text has been formatted.
// It outputs a closing `</span>` if necessary, and a closing `</pre>` tag.
// (Part of interface Format)
func (html *HTML) Postamble(w io.Writer) {
	if html.bidi {
		w.Write([]byte("</span>"))
	}
	w.Write([]byte("\n<pre>\n"))
}

// LTR signals to w that a bidi.LeftToRight sequence is to be output.
// It outputs a closing `</span>` if necessary, and a `<span dir="ltr">` tag.
// (Part of interface Format)
func (html *HTML) LTR(w io.Writer) {
	if html.bidi {
		w.Write([]byte("</span>"))
	}
	w.Write([]byte("<span dir=\"ltr\">"))
	html.dir = bidi.LeftToRight
	html.bidi = true
}

// RTL signals to w that a bidi.RightToLeft sequence is to be output.
// It outputs a closing `</span>` if necessary, and a `<span dir="rtl">` tag.
// (Part of interface Format)
func (html *HTML) RTL(w io.Writer) {
	if html.bidi {
		w.Write([]byte("</span>"))
	}
	w.Write([]byte("<span dir=\"rtl\">"))
	html.dir = bidi.RightToLeft
	html.bidi = true
}

// Line is a signal from the output driver that a new line is to be output.
// length is the total width of the characters that will be formatted, measured
// in “en”s, i.e. fixed width positions. linelength is the target line length
// to wrap long lines.
//
// Currently does nothing.
// (Part of interface Format)
func (html *HTML) Line(length int, linelength int, w io.Writer) {
}

// Newline will be called at the end of every formatted line of text.
// It outputs a `<br>` tag.
// (Part of interface Format)
func (html *HTML) Newline(w io.Writer) {
	w.Write([]byte("<br>"))
}

// NeedsReordering signals to the formatting driver what kind of support the
// console needs with Bidi text.
func (html *HTML) NeedsReordering() ReorderFlag {
	return html.reorderPolicy
}

// HTMLStyle is a style equivalent to inline.Style, which offers some convenience
// functions.
type HTMLStyle inline.Style

func (s HTMLStyle) String() string {
	return s.tags(false)
}

// Equals is part of interface styled.Style.
func (s HTMLStyle) Equals(other styled.Style) bool {
	return inline.Style(s).Equals(other)
}

func (s HTMLStyle) tags(closing bool) string {
	if s == 0 {
		return ""
	}
	str := ""
	if closing {
		for i := 6; i >= 0; i-- {
			//T().Debugf("check: %d = %s", 1<<i, styleString(1<<i))
			if s&(1<<i) > 0 {
				str = str + "</" + htmlStyleNames[1<<i] + ">"
			}
		}
	} else {
		for i := 0; i < 7; i++ {
			//T().Debugf("check: %d = %s", 1<<i, styleString(1<<i))
			if s&(1<<i) > 0 {
				str = str + "<" + htmlStyleNames[1<<i] + ">"
			}
		}
	}
	return str // may be empty string
}

// Add combines a style with another style
func (s HTMLStyle) Add(sty HTMLStyle) HTMLStyle {
	return HTMLStyle(s.Add(sty))
}

/*
// HTMLter formats a styled text as HTML
type HTMLter struct {
	out    *bytes.Buffer
	suffix string
}

func NewHTMLter(prefix, suffix string) *HTMLter {
	return &HTMLter{
		out:    bytes.NewBufferString(prefix),
		suffix: suffix,
	}
}

func (fmtr HTMLter) String() string {
	return fmtr.out.String() + fmtr.suffix
}

func (fmtr HTMLter) Writer() io.Writer {
	return fmtr.out
}

func (fmtr HTMLter) StartRun(f styled.Style, w io.Writer) error {
	if f == nil {
		return nil
	}
	var hsty HTMLStyle
	if sty, ok := f.(HTMLStyle); ok {
		hsty = sty
	} else if sty, ok := f.(Style); ok {
		hsty = HTMLStyle{sty}
	} else {
		return nil
	}
	_, err := w.Write([]byte(hsty.String()))
	return err
}

func (fmtr HTMLter) Format(buf []byte, f styled.Style, w io.Writer) error {
	w.Write(buf)
	return nil
}

func (fmtr HTMLter) EndRun(f styled.Style, w io.Writer) error {
	if f == nil {
		return nil
	}
	var hsty HTMLStyle
	if sty, ok := f.(HTMLStyle); ok {
		hsty = sty
	} else if sty, ok := f.(Style); ok {
		hsty = HTMLStyle{sty}
	} else {
		return nil
	}
	_, err := w.Write([]byte(hsty.tags(true)))
	return err
}
*/
