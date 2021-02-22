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
	"github.com/npillmayer/cords/styled/inline"
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

/*
type HTMLStyle inline.Style

func (s HTMLStyle) String() string {
	return s.tags(false)
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

func (s HTMLStyle) Add(sty HTMLStyle) HTMLStyle {
	return HTMLStyle(s.Add(sty))
}

// HTMLFormatter formats a styled text as HTML
type HTMLFormatter struct {
	out    *bytes.Buffer
	suffix string
}

func NewHTMLFormatter(prefix, suffix string) *HTMLFormatter {
	return &HTMLFormatter{
		out:    bytes.NewBufferString(prefix),
		suffix: suffix,
	}
}

func (fmtr HTMLFormatter) String() string {
	return fmtr.out.String() + fmtr.suffix
}

func (fmtr HTMLFormatter) Writer() io.Writer {
	return fmtr.out
}

func (fmtr HTMLFormatter) StartRun(f styled.Style, w io.Writer) error {
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

func (fmtr HTMLFormatter) Format(buf []byte, f styled.Style, w io.Writer) error {
	w.Write(buf)
	return nil
}

func (fmtr HTMLFormatter) EndRun(f styled.Style, w io.Writer) error {
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
