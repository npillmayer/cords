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

// Example for bi-directional text and line-breaking according to the Unicode
// Bidi algorithm. We set up an unusual console format to make newlines visible
// in the Godoc documentation. Then we configure for a line length of 40 'en's,
// which will ensure a line-break between the two words in hebrew script.
//
// Please note that this is in a sense a contrieved example, as it has to work
// from godoc in the browser. The browser will do the right thing with Bidi anyway.
// However, the example shows a typical use case and has a chance to work on
// different terminals with varying support for bidi text.
func TestConsoleFixedWidth(t *testing.T) {
	//func ExampleConsoleFixedWidth() {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()
	//
	console := NewLocalConsoleFormat()
	console.Codes.Newline = []byte("<nl>\n") // just to please godoc
	config := &Config{LineWidth: 40}         // format into narrow lines
	//
	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	para, _ := styled.ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	console.Print(para, config)
	// Output:
	// The quick brown fox jumps over the כלב <nl>
	// עצלן!<nl>
	//
	t.Fail()
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
