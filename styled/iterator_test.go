package styled

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/uax/bidi"
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

func TestTextBidiStyleRangesSimple(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.styles")
	defer teardown()

	data := []struct {
		input  string
		runcnt int
	}{
		{"The quick brown fox jumps over the lazy dog", 3},
		{
			"Look out: זה כיף סתם לשמוע איך תנצח קרפד עץ טוב בגן",
			4,
		}, {"Look : זה כיף סתם לשמוע איך תנצח קרפד עץ טוב בגן",
			4, // bold overlaps RTL
		}, {"Ciao היי World!", 4}, // split mid-RTL
	}
	p := 0
	for _, c := range data[3].input {
		l := utf8.RuneLen(c)
		t.Logf(" char[%2d]='%c' code=%v  len=%d", p, c, c, l)
		p += l
	}
	//t.Fatal()
	bold := teststyle("bold")
	var err error
	for _, d := range data {
		text := TextFromString(d.input)
		if text, err = text.Style(bold, 5, 9); err != nil {
			t.Fatalf("initial styling failed: %v", err)
		}
		para, err := ParagraphFromText(&text, 0, text.Raw().Len(), bidi.LeftToRight, nil)
		if err != nil {
			t.Fatalf("ParagraphFromText failed: %v", err)
		}
		t.Log("-----------------------------")
		count := 0
		for rnge, reader := range para.BidiStyleRanges() {
			s, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("io.ReadAll failed: %v", err)
			}
			t.Logf("style range %d: %v", count, rnge)
			t.Logf("       text %d: “%s”", count, s)
			count++
		}
		if count != d.runcnt {
			t.Fatalf("expected %d style ranges, got %d", d.runcnt, count)
		}
		t.Log("==========================================================")
	}
	//t.Fail()
}
