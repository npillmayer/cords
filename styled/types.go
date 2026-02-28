package styled

import (
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/cordext"
)

// ---------------------------------------------------------------------------
type Run struct {
	length uint64
	style  Style
}

func (run Run) Summary() Summary {
	return Summary{runs: []Run{run}}
}

type Summary struct {
	runs []Run
}

func (s Summary) length() uint64 {
	var l uint64
	for _, r := range s.runs {
		l += r.length
	}
	return l
}

type monoid struct{}

func (m monoid) Zero() Summary {
	return Summary{runs: []Run{}}
}

func (m monoid) Add(left, right Summary) Summary {
	r := merge(left.runs, right.runs)
	return Summary{r}
}

// ---------------------------------------------------------------------------

type Runs struct {
	tree *btree.Tree[Run, Summary, btree.NO_EXT]
}

func newRuns() (Runs, error) {
	var config = btree.Config[Run, Summary, btree.NO_EXT]{
		Monoid:    monoid{},
		Extension: nil,
	}
	tree, err := btree.New(config)
	if err != nil {
		return Runs{}, err
	}
	runs := Runs{tree}
	return runs, err
}

// ---------------------------------------------------------------------------

// Text is a styled text. Its text and its styles are automatically synchronized.
type Text struct {
	text cordext.CordEx[btree.NO_EXT] // TODO rename to `raw`
	runs Runs
}

func (t Text) isUnstyled() bool {
	return t.runs.tree == nil || t.runs.tree.IsEmpty()
}

// ---------------------------------------------------------------------------

type styleDimension struct{}

func (styleDimension) Zero() uint64 { return 0 }

func (styleDimension) Add(acc uint64, summary Summary) uint64 {
	return acc + summary.length()
}

func (styleDimension) Compare(acc uint64, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}
