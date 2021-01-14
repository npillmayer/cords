package inline

import (
	"bytes"
	"io"

	"github.com/npillmayer/cords/styled"
)

var htmlStyleNames map[Style]string = map[Style]string{
	PlainStyle:   "",
	BoldStyle:    "b",
	ItalicsStyle: "i",
	StrongStyle:  "strong",
	EmStyle:      "em",
	SmallStyle:   "small",
	MarkedStyle:  "marked",
}

type HTMLStyle struct {
	Style
}

func (s HTMLStyle) String() string {
	return s.tags(false)
}

func (s HTMLStyle) tags(closing bool) string {
	if s.Style == 0 {
		return ""
	}
	str := ""
	if closing {
		for i := 6; i >= 0; i-- {
			//T().Debugf("check: %d = %s", 1<<i, styleString(1<<i))
			if s.Style&(1<<i) > 0 {
				str = str + "</" + styleString(1<<i) + ">"
			}
		}
	} else {
		for i := 0; i < 7; i++ {
			//T().Debugf("check: %d = %s", 1<<i, styleString(1<<i))
			if s.Style&(1<<i) > 0 {
				str = str + "<" + styleString(1<<i) + ">"
			}
		}
	}
	return str // may be empty string
}

func (s HTMLStyle) Add(sty Style) HTMLStyle {
	return HTMLStyle{s.Style.Add(sty)}
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

func (fmtr HTMLFormatter) StartRun(f styled.Format, w io.Writer) error {
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

func (fmtr HTMLFormatter) Format(buf []byte, f styled.Format, w io.Writer) error {
	w.Write(buf)
	return nil
}

func (fmtr HTMLFormatter) EndRun(f styled.Format, w io.Writer) error {
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
