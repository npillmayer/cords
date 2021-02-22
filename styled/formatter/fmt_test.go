package formatter

import (
	"testing"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/inline"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/bidi"
)

func TestFmt1(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	//text := styled.TextFromString("The quick brown fox jumps over the lazy dog!")
	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	text.Style(inline.BoldStyle, 4, 9)
	para, err := styled.ParagraphFromText(text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	console := NewConsoleFixedWidthFormat(nil, nil)
	err = console.Print(para, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	//t.Fail()
}

// Example for bi-directional text and line-breaking according to the Unicode
// Bidi algorithm. We set up an unusual console format to make newlines visible
// in the Godoc documentation. Then we configure for a line length of 40 'en's,
// which will ensure a line-break between the two words in hebrew script.
func ExampleConsoleFixedWidth() {
	console := NewConsoleFixedWidthFormat(&ControlCodes{Newline: []byte("<nl>\n")}, nil)
	config := &Config{LineWidth: 40}
	//
	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	para, _ := styled.ParagraphFromText(text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	console.Print(para, config)
	// Output:
	// The quick brown fox jumps over the כלב <nl>
	// עצלן!<nl>
	//
}
