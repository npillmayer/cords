package inline

import (
	"io"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestTextSimple(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t)
	defer teardown()
	//
	s := ItalicsStyle
	t.Logf("italics=%v", styleString(s))
	s = s.Add(MarkedStyle)
	t.Logf("combined=%v", s.String())
}

func TestHTMLSimple(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t)
	defer teardown()
	//
	input := strings.NewReader("The quick <strong>brown</strong> fox <em>jumps</em> over the lazy dog")
	text, err := TextFromHTML(input)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("HTML inner text = '%s'", text.Raw())
	cnt := 0
	for run, reader := range text.StyleRanges() {
		s, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Logf("  %d: %v", run, string(s))
		cnt++
	}
	if cnt != 5 {
		t.Errorf("expected 5 style runs, got %d", cnt)
	}
}
