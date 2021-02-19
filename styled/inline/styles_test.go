package inline

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	//"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestTextSimple(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gotestingadapter.New()
	// teardown := gotestingadapter.RedirectTracing(t)
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
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gotestingadapter.New()
	// teardown := gotestingadapter.RedirectTracing(t)
	// defer teardown()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	s := HTMLStyle{ItalicsStyle}
	t.Logf("italics=%s", s)
	s = s.Add(MarkedStyle)
	t.Logf("combined=%v", s.String())
	//t.Fail()
}

/*
func TestHTMLFormatter(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	text := styled.TextFromString("Hello World, how are you?")
	bold, italic := HTMLStyle{BoldStyle}, HTMLStyle{ItalicsStyle}
	text.Style(bold, 6, 11)
	text.Style(italic, 13, 16) // erase part of bold run
	fmtr := NewHTMLFormatter("<html><body><p>", "</p></body></html>")
	if err := text.Format(fmtr, fmtr.Writer()); err != nil {
		t.Error(err.Error())
	}
	t.Logf(fmtr.String())
	//t.Fail()
}
*/
