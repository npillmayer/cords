package formatter

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

/*
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
