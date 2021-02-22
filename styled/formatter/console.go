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
	"os"

	"github.com/fatih/color"
	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/inline"
	"github.com/npillmayer/uax/uax11"
	"golang.org/x/term"
)

// ControlCodes holds certain escape sequences which a terminal uses to control
// Bidi behaviour.
type ControlCodes struct {
	Preamble, Postamble []byte
	LTR, RTL            []byte
	Newline             []byte
}

// DefaultCodes is the default set of control codes.
// See https://terminal-wg.pages.freedesktop.org/bidi/recommendation/escape-sequences.html
//
var DefaultCodes = ControlCodes{
	Preamble:  []byte{27, '[', '8', 'l'}, // switch to explicit mode
	Postamble: []byte{},
	LTR:       []byte{27, '[', '1', ' ', 'k'},
	RTL:       []byte{27, '[', '2', ' ', 'k'},
	Newline:   []byte{'\n'},
	//Default: CSI 0 SPACE k (or CSI SPACE k)
}

// ConsoleFixedWidth is a type for outputting formatted text to a console with
// a fixed width font.
//
// Console/Terminal output is notoriously tricky for bi-directional text and for
// scripts other than Latin. To fully appreciate the difficulties behind this,
// refer for example to
// https://terminal-wg.pages.freedesktop.org/bidi/bidi-intro/why-terminals-are-special.html
//
// As long as there is not widely accepted standard for Bidi-handling in terminals, we
// have to rely on heuristics and explicitly set device-dependent configuration.
// This is unfortunate for applications which are supposed to run in multi-platform
// and multi-regional environments. However, it is no longer acceptable for
// applications to be content with handling Latin text only.
//
type ConsoleFixedWidth struct {
	Codes   *ControlCodes
	colors  map[styled.Style]*color.Color
	ccnt    int // number of character positions already printed for line
	ctarget int // linelength in fixedwidth ‘en’s
}

// Print outputs a styled paragraph to stdout.
//
// If parameter config is nil,
// a heuristic will create a config from the current terminal's properties (if
// stdout is interactive). Config.Context will also be created based on heuristics
// from the user environment.
func (fw *ConsoleFixedWidth) Print(para *styled.Paragraph, config *Config) error {
	if config == nil {
		config = ConfigFromTerminal()
		config.Context = uax11.ContextFromEnvironment()
	}
	return Output(para, os.Stdout, config, fw)
}

// NewConsoleFixedWidthFormat creates a new formatter. It is to be used for consoles
// with a fixed width font.
//
// codes is a table of escape sequences to control Bidi behaviour of the console.
// colors is a map from the styled.Styles to colors, used for display. It may contain
// just a subset of the styles used in the texts which will be handled
// by this formatter.
//
func NewConsoleFixedWidthFormat(codes *ControlCodes, colors map[styled.Style]*color.Color) *ConsoleFixedWidth {
	fw := &ConsoleFixedWidth{
		Codes: &DefaultCodes,
	}
	if codes != nil {
		fw.Codes = codes
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

// StyledText is called by the formatting driver to output a sequence of
// uniformly styled text (item). It uses colors to visualize styles.
// (Part of interface Format)
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

// Preamble is called by the output driver before a paragraph of text will be formatted.
// It outputs the `Preamble` escape sequence from fw.Codes.
// (Part of interface Format)
func (fw *ConsoleFixedWidth) Preamble(w io.Writer) {
	w.Write(fw.Codes.Preamble)
}

// Postamble will be called after a paragraph of text has been formatted.
// It outputs the `Postamble` escape sequence from fw.Codes.
// (Part of interface Format)
func (fw *ConsoleFixedWidth) Postamble(w io.Writer) {
	w.Write(fw.Codes.Preamble)
}

// LTR signals to w that a bidi.LeftToRight sequence is to be output.
// (Part of interface Format)
func (fw *ConsoleFixedWidth) LTR(w io.Writer) {
	w.Write(fw.Codes.LTR)
}

// RTL signals to w that a bidi.RightToLeft sequence is to be output.
// (Part of interface Format)
func (fw *ConsoleFixedWidth) RTL(w io.Writer) {
	w.Write(fw.Codes.RTL)
}

// Line is a signal from the output driver that a new line is to be output.
// length is the total width of the characters that will be formatted, measured
// in “en”s, i.e. fixed width positions. linelength is the target line length
// to wrap long lines.
// (Part of interface Format)
func (fw *ConsoleFixedWidth) Line(length int, linelength int, w io.Writer) {
	fw.ccnt = 0
	fw.ctarget = linelength
}

// Newline will be called at the end of every formatted line of text.
// It outputs the `Newline` escape sequence from fw.Codes.
// (Part of interface Format)
func (fw *ConsoleFixedWidth) Newline(w io.Writer) {
	w.Write(fw.Codes.Newline)
}

// --- Config for terminals --------------------------------------------------

// ConfigFromTerminal is a simple helper for creating a formatting Config.
// It checks wether stdout is a terminal, and if so it reads the terminal's width
// and sets the Config.LineWidth parameter accordingly.
func ConfigFromTerminal() *Config {
	config := &Config{}
	if term.IsTerminal(0) {
		w, _, err := term.GetSize(0)
		if err != nil {
			config.LineWidth = 65
		} else {
			if w > 65 {
				config.LineWidth = w - 10
			} else if w > 30 {
				config.LineWidth = w - 5
			} else if w > 10 {
				config.LineWidth = w
			} else {
				config.LineWidth = 10
			}
		}
	} else {
		config.LineWidth = 65
	}
	T().P("format", "console").Infof("setting line length to %d en", config.LineWidth)
	return config
}
