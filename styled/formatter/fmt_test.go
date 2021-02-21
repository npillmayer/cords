package formatter

import (
	"testing"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/inline"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/bidi"
	"github.com/npillmayer/uax/grapheme"
)

func TestFmt1(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	grapheme.SetupGraphemeClasses()
	//text := styled.TextFromString("The quick brown fox jumps over the lazy dog!")
	text := styled.TextFromString("The quick brown fox jumps over the כלב עצלן!")
	text.Style(inline.BoldStyle, 4, 9)
	para, err := styled.ParagraphFromText(text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	err = Print(para, nil)
	t.Fail()
}
