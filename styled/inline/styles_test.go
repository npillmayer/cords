package inline

import (
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
)

func TestTextSimple(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// teardown := gotestconfig.QuickConfig(t)
	// defer teardown()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	s := ItalicsStyle
	t.Logf("italics=%v", styleString(s))
	s = s.Add(MarkedStyle)
	t.Logf("combined=%v", s.String())
	//t.Fail()
}

func TestHTMLSimple(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	// teardown := gotestconfig.QuickConfig(t)
	// defer teardown()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	input := strings.NewReader("The quick <strong>brown</strong> fox <em>jumps</em> over the lazy dog")
	text, err := TextFromHTML(input)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("HTML inner text = '%s'", text.Raw())
	styles := text.StyleRuns()
	cnt := 0
	for i, style := range styles {
		t.Logf("  %d: %v", i, style)
		cnt++
	}
	if cnt != 5 {
		t.Errorf("expected 5 style runs, got %d", cnt)
	}
}
