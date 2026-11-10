package styled

import (
	"errors"
	"testing"
)

func TestEmptyStyleSpanPreservesTextAndRuns(t *testing.T) {
	for _, raw := range []string{"", "é世 abc"} {
		for _, styledAlready := range []bool{false, true} {
			text := TextFromString(raw)
			if styledAlready && len(raw) > 0 {
				var err error
				text, err = text.Style(teststyle("original"), 0, 2)
				if err != nil {
					t.Fatal(err)
				}
			}
			for p := uint64(0); p <= uint64(len(raw)); p++ {
				for _, sty := range []Style{nil, teststyle("new")} {
					got, err := text.Style(sty, p, p)
					if err != nil {
						t.Fatalf("raw=%q styled=%v [%d,%d): %v", raw, styledAlready, p, p, err)
					}
					if got.Raw().String() != raw || got.Raw().Len() != text.Raw().Len() || got.runs.tree != text.runs.tree {
						t.Fatal("empty span changed text or allocated/replaced runs")
					}
					runs, err := text.runs.Style(uint64(len(raw)), sty, p, p)
					if err != nil || runs.tree != text.runs.tree {
						t.Fatalf("Runs.Style empty range changed runs: %v", err)
					}
					initial, err := initialStyle(uint64(len(raw)), sty, p, p)
					if err != nil || initial.tree != nil {
						t.Fatalf("initialStyle empty range allocated runs: %v", err)
					}
				}
			}
		}
	}
}

func TestStyleEmptySpanDoesNotAcceptInvalidRanges(t *testing.T) {
	for _, raw := range []string{"", "é世 abc"} {
		n := uint64(len(raw))
		for _, already := range []bool{false, true} {
			text := TextFromString(raw)
			if already && n > 0 {
				var err error
				text, err = text.Style(teststyle("original"), 0, n)
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, span := range [][2]uint64{{n + 1, n + 1}, {n + 1, n + 2}, {n + 1, 0}, {n, n + 1}} {
				got, err := text.Style(nil, span[0], span[1])
				if !errors.Is(err, ErrIllegalArguments) || got.runs.tree != text.runs.tree || got.Raw().String() != raw {
					t.Fatalf("range %v: expected error and unchanged text, got %v", span, err)
				}
				runs, err := text.runs.Style(n, nil, span[0], span[1])
				if !errors.Is(err, ErrIllegalArguments) || runs.tree != text.runs.tree {
					t.Fatalf("Runs.Style range %v: %v", span, err)
				}
			}
		}
	}
}

func TestStyleNonemptyEndStillClamps(t *testing.T) {
	for _, already := range []bool{false, true} {
		text := TextFromString("éabc")
		var err error
		if already {
			text, err = text.Style(teststyle("old"), 0, 5)
			if err != nil {
				t.Fatal(err)
			}
		}
		got, err := text.Style(teststyle("tail"), 2, 100)
		if err != nil {
			t.Fatal(err)
		}
		for p := uint64(2); p < 5; p++ {
			s, _, err := got.StyleAt(p)
			if err != nil || !equals(s, teststyle("tail")) {
				t.Fatal("clamped style lost")
			}
		}
	}
}
