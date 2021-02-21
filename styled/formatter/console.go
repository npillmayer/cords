package formatter

import (
	"io"

	"github.com/fatih/color"
	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/inline"
)

type ControlCodes struct {
	Preamble, Postamble []byte
	LTR, RTL            []byte
	Newline             []byte
}

// DefaultCodes is the default set of control codes.
// See https://terminal-wg.pages.freedesktop.org/bidi/recommendation/escape-sequences.html#ltr-vs-rtl
//
var DefaultCodes = ControlCodes{
	Preamble:  []byte{27, '[', '8', 'l'}, // switch to explicit mode
	Postamble: []byte{},
	LTR:       []byte{27, '[', '1', ' ', 'k'},
	RTL:       []byte{27, '[', '2', ' ', 'k'},
	Newline:   []byte{'\n'},
	//Default: CSI 0 SPACE k (or CSI SPACE k)
}

type ConsoleFixedWidth struct {
	codes   *ControlCodes
	colors  map[styled.Style]*color.Color
	ccnt    int // number of character positions already printed for line
	ctarget int // linelength in fixedwidth ‘en’s
}

func NewConsoleFixedWidthFormat(codes *ControlCodes, colors map[styled.Style]*color.Color) *ConsoleFixedWidth {
	fw := &ConsoleFixedWidth{
		codes: &DefaultCodes,
	}
	if codes != nil {
		fw.codes = codes
	}
	if colors == nil {
		fw.colors = makeDefaultPalette()
	} else {
		fw.colors = colors
	}
	return fw
}

func makeDefaultPalette() map[styled.Style]*color.Color {
	palette := map[styled.Style]*color.Color{
		inline.PlainStyle: color.New(color.FgBlue),
		inline.BoldStyle:  color.New(color.FgRed),
	}
	return palette
}

func (fw *ConsoleFixedWidth) StyledText(s string, style styled.Style, w io.Writer) {
	if style != nil {
		c, ok := fw.colors[style]
		if ok {
			c.Fprint(w, s)
			return
		}
	}
	w.Write([]byte(s))
}

func (fw *ConsoleFixedWidth) Preamble(w io.Writer) {
	w.Write(fw.codes.Preamble)
}

func (fw *ConsoleFixedWidth) Postamble(w io.Writer) {
	w.Write(fw.codes.Preamble)
}

func (fw *ConsoleFixedWidth) LTR(w io.Writer) {
	w.Write(fw.codes.LTR)
}

func (fw *ConsoleFixedWidth) RTL(w io.Writer) {
	w.Write(fw.codes.RTL)
}

func (fw *ConsoleFixedWidth) Line(length int, linelength int, w io.Writer) {
	fw.ccnt = 0
	fw.ctarget = linelength
}

func (fw *ConsoleFixedWidth) Newline(w io.Writer) {
	w.Write(fw.codes.Newline)
}
