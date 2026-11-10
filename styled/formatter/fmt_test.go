package formatter

import (
	"bytes"
	"os"
	"testing"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/inline"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/uax/bidi"
)

func TestReorderGraphemes(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()
	//
	s := "Hello 👍🏼!"
	s = reorder(s, ReorderGraphemes)
	t.Logf("s = '%s'", s)
	if s != "!👍🏼 olleH" {
		t.Error("expected string to be reversed by graphemes, isn't: ", s)
	}
}

func TestFmt1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()
	//
	//text := styled.TextFromString("The quick brown fox jumps over the lazy dog!")
	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	t.Logf("text length = %d", text.Raw().Len())
	text.Style(inline.BoldStyle, 4, 9)
	para, err := styled.ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	console := NewConsoleFixedWidthFormat(nil, nil, ReorderWords)
	err = console.Print(para, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestVTE(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()
	//
	//text := styled.TextFromString("The quick brown fox jumps over the lazy dog!")
	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	text.Style(inline.BoldStyle, 4, 9)
	para, err := styled.ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	os.Setenv("VTE_VERSION", "123")
	console := newXTermFormat("xterm-color256")
	err = console.Print(para, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	//t.Errorf("TODO: RTL does not work")
}

// Exercise fixed-width output with explicit device settings, independent of
// TERM, VTE_VERSION, stdout, and the host terminal's bidi capabilities.
func TestConsoleFixedWidth(t *testing.T) {
	const source = "The quick brown fox jumps over the כלב עצלן!"
	for _, tc := range []struct {
		name  string
		width int
		want  string
	}{
		{"wrap between Hebrew words", 40, "The quick brown fox jumps over the כלב <nl>\nעצלן!<nl>\n"},
		{"wide line", 80, source + "<nl>\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Output consumes the paragraph via WrapAt, so each case needs a fresh one.
			text := styled.TextFromString(source)
			para, err := styled.ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
			if err != nil {
				t.Fatal(err)
			}
			// ReorderNone leaves Hebrew in logical order for a bidi-capable device.
			// A local codes value avoids mutating the package-wide defaults.
			codes := ControlCodes{Newline: []byte("<nl>\n")}
			console := NewConsoleFixedWidthFormat(&codes, nil, ReorderNone)
			var out bytes.Buffer
			if err := Output(para, &out, &Config{LineWidth: tc.width}, console); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("width %d: output = %q, want %q", tc.width, got, tc.want)
			}
		})
	}
}

func TestHTML1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()
	//
	var err error
	//text := styled.TextFromString("The quick brown fox jumps over the lazy dog!")
	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	text, err = text.Style(inline.BoldStyle, 4, 9)
	para, err := styled.ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	var b bytes.Buffer
	html := NewHTML(ReorderNone)
	html.Print(para, &b, nil)
	t.Logf("HTML output =\n'%s'", b.String())
}
