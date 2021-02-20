package formatter

import (
	"os"
	"testing"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/cords/styled/inline"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/uax/bidi"
	"github.com/npillmayer/uax/grapheme"
	"github.com/npillmayer/uax/uax11"
)

func TestFmt1(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	grapheme.SetupGraphemeClasses()
	text := styled.TextFromString("The quick brown fox jumps over the lazy dog!")
	text.Style(inline.BoldStyle, 5, 10)
	para, err := styled.ParagraphFromText(text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	config := &Config{
		LineWidth: 30,
		Context:   uax11.LatinContext,
	}
	err = Format(para, os.Stdout, config)
	t.Fail()
}
