/*
Package formatter formats styled text on output devices with
fixed-width fonts. It is intended for situations where the application is
responsible for the visual representation (as opposed to output to a
browser, which usually addresses the complications of text by itself,
transparently for applications).
Think of this package in terms of `fmt.Println` for styled, bi-directional
text.

Output of styled text differs in many aspectes from simple string output.
Not only do we need an output device which is capable of displaying text
styles, but we need to consider line-breaking and the handling of
bi-directional (Bidi) text as well. This package helps performing the
following tasks:

▪︎ Select a formatter for a given (monospaced) output device

▪︎ Create a suitable formatting configuration

▪︎ Format a styled paragraph of possibly bi-directional text and output it to the device

Formatting and output needs to perform a couple of steps to produce a
correct visual representation. These steps are in a large part covered
by various Unicode Annexes and in general it's non-trivial to get them
right (https://raphlinus.github.io/text/2020/10/26/text-layout.html).
Package formatter will apply rules from UAX#9 (bidi), UAX#14 (line breaking),
UAX#29 (graphemes) and UAX#11 (character width), as well as some heuristics
depending on the output device.

This package does not constitute a typesetter. We will
not deal with fonts, glyphing, variable text widths, elaborate line-breaking algorithms,
etc. In particular we will not handle issues having to do with fonts or with
locale-specific glyphs missing for an output device.

The Problems it Solves

As an application developer most of the time you do not have a need to consider
the fine points of styled and bidirectional text. Most applications deal with
strings, not text
(https://mortoray.com/2014/03/17/strings-and-text-are-not-the-same/).

However, if you happen to really need it, support for text as a data structure is
sparse in system developement languages like Go (Rust is about to prove me wrong on this),
and dealing with bidi text is sometimes complicated. What's more:
libraries for text have peculiar problems during test, as there is no easy
output target, except browsers and terminals. And browsers are – of all applications –
among the best when dealing with text styles and bidi. That makes it sometimes hard
to test your own bidi- or styling algorithms, as it will interfere with the
browser logic. And terminals have their own kinds of challenges with bidi, making
it often difficult to pinpoint an error.

API

Clients select an instance of type formatter.Format and possibly configure it
to their needs. As soon as a piece of styled text is to be output, it has to
be broken up into paragraphs. This is due to the fact that the Unicode Bidi
Algorithm works on paragraphs. Breaking up into paragraphs may be done by the
client explicitely, or a formatter may be able to do the paragraph-splitting itself.

	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	text.Style(inline.BoldStyle, 4, 9)  // want 'quick' in boldface
	para, _ := styled.ParagraphFromText(text, 0, text.Raw().Len(), bidi.LeftToRight, nil)

	console := NewLocalConsoleFormat()
	console.Print(para, nil)

formatter.Format is an interface type and this package offers two implementations,
one for console output (like in the example above) and one for HTML output.

Status

Work in progress, especially the HTML formatter is in it's infancy.
Needs a lot more testing.
API not stable.

_________________________________________________________________________

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
package formatter

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to a global core-tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
}
