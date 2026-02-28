package styled

import (
	"io"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestTextStyleRangesSimple(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	text := TextFromString("The quick brown fox jumps over the lazy dog")
	bold := teststyle("bold")
	var err error
	if text, err = text.Style(bold, 4, 15); err != nil {
		t.Fatalf("initial styling failed: %v", err)
	}
	count := 0
	for rnge, _ := range text.StyleRanges() {
		t.Logf("style range %d: %v", count, rnge)
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 style ranges, got %d", count)
	}
}

func TestTextStyleRangesWithReader(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	inptext := "The quick brown fox jumps over the lazy dog"
	text := TextFromString(inptext)
	bold := teststyle("bold")
	var err error
	if text, err = text.Style(bold, 4, 15); err != nil {
		t.Fatalf("initial styling failed: %v", err)
	}
	var frags []string
	count := 0
	for rnge, reader := range text.StyleRanges() {
		if reader == nil {
			t.Errorf("expected reader to be non-nil")
		}
		s, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("error reading from reader: %v", err)
		}
		t.Logf("range %d: %v = '%s'", count, rnge, string(s))
		frags = append(frags, string(s))
		count++
	}
	if count != 3 {
		t.Errorf("expected 3 style ranges, got %d", count)
	}
	var sb strings.Builder
	for i := range count {
		sb.WriteString(frags[i])
	}
	if sb.String() != inptext {
		t.Errorf("expected text '%s', got '%s'", inptext, sb.String())
	}
}
