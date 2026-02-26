package styled

import (
	"github.com/npillmayer/cords/btree"
)

// SplitAt splits style runs at byte position pos, returning left and right
// partitions for ranges [0,pos) and [pos,total).
func (runs Runs) SplitAt(pos uint64) (Runs, Runs, error) {
	//mkEmpty := func() (Runs, error) { return newRuns() }
	p := pipeFor(runs)
	if p.err != nil {
		if pos != 0 {
			return Runs{}, Runs{}, ErrIndexOutOfBounds
		}
		p.err = nil
		p.runs = pipeCall(p, newRuns)
		l := p.runsOrElse(Runs{})
		p.runs = pipeCall(p, newRuns)
		r := p.runsOrElse(Runs{})
		return l, r, p.err
	}
	// if runs.tree == nil || runs.tree.IsEmpty() {
	// 	if pos != 0 {
	// 		return Runs{}, Runs{}, cords.ErrIndexOutOfBounds
	// 	}
	// 	left, err := mkEmpty()
	// 	if err != nil {
	// 		return Runs{}, Runs{}, err
	// 	}
	// 	right, err := mkEmpty()
	// 	if err != nil {
	// 		return Runs{}, Runs{}, err
	// 	}
	// 	return left, right, nil
	// }
	total := runs.tree.Summary().length()
	if pos > total {
		return Runs{}, Runs{}, ErrIndexOutOfBounds
	}
	if pos == 0 {
		p.runs = pipeCall(p, newRuns)
		return p.runsOrElse(Runs{}), runs, p.err
		// left, err := mkEmpty()
		// if err != nil {
		// 	return Runs{}, Runs{}, err
		// }
		// return left, runs, nil
	}
	if pos == total {
		p.runs = pipeCall(p, newRuns)
		return runs, p.runsOrElse(Runs{}), p.err
		// right, err := mkEmpty()
		// if err != nil {
		// 	return Runs{}, Runs{}, err
		// }
		// return runs, right, nil
	}

	tree := runs.tree
	cursor, err := btree.NewCursor(tree, styleDimension{})
	if err != nil {
		return Runs{}, Runs{}, err
	}
	idx, run, runStart, runEnd, err := seekRunForByte(tree, cursor, pos-1)
	if err != nil {
		return Runs{}, Runs{}, err
	}
	if pos > runStart && pos < runEnd {
		leftFrag := Run{length: pos - runStart, style: run.style}
		rightFrag := Run{length: runEnd - pos, style: run.style}
		ins2 := func(i int64, l, r Run) (*btree.Tree[Run, Summary, btree.NO_EXT], error) {
			return tree.InsertAt(i, l, r)
		}
		tree = pipeCall2(p, tree.DeleteRange, idx, 1)
		tree = pipeCall2a(p, ins2, idx, leftFrag, rightFrag)
		// if p.err != nil {
		// 	return Runs{}, Runs{}, p.err
		// }
		// ----------------------------------------
		// tree, err = tree.DeleteRange(idx, 1)
		// if err != nil {
		// 	return Runs{}, Runs{}, err
		// }
		// tree, err = tree.InsertAt(idx, leftFrag, rightFrag)
		// if err != nil {
		// 	return Runs{}, Runs{}, err
		// }
	}
	left, right := pipeCall1to2(p, tree.SplitAt, idx+1)
	if p.err != nil {
		return Runs{}, Runs{}, err
	}
	return Runs{tree: left}, Runs{tree: right}, nil

	// leftTree, rightTree, err := tree.SplitAt(idx + 1)
	// if err != nil {
	// 	return Runs{}, Runs{}, err
	// }
	// return Runs{tree: leftTree}, Runs{tree: rightTree}, nil
}

// Concat concatenates two run sets and repairs the seam to preserve the run
// invariant (no adjacent equal styles).
func (runs Runs) Concat(other Runs) (Runs, error) {
	if p_runs := pipeFor(runs); p_runs.err != nil {
		return other, nil
	}
	p_other := pipeFor(other)
	if p_other.err != nil {
		return runs, nil
	}
	// ---------
	// if other.tree == nil || other.tree.IsEmpty() {
	// 	return runs, nil
	// }
	leftLen := runs.tree.Len()
	tree := pipeCall1(p_other, runs.tree.Concat, other.tree)
	// tree, err := runs.tree.Concat(other.tree)
	// if err != nil {
	// 	return Runs{}, err
	// }
	p_other.runs.tree, _ = pipeCall2to2(p_other, mergeAdjacentRuns, tree, leftLen-1)
	r := p_other.runsOrElse(Runs{})
	return r, p_other.err
	// if tree, _, err = mergeAdjacentRuns(tree, leftLen-1); err != nil {
	// 	return Runs{}, err
	// }
	// return Runs{tree: tree}, nil
}

// Section returns the sub-range of runs covering [from,to).
func (runs Runs) Section(from, to uint64) (Runs, error) {
	// mkEmpty := func() (Runs, error) {
	// 	return newRuns()
	// }
	// if from > to {
	// 	return Runs{}, cords.ErrIndexOutOfBounds
	// }
	p := pipeFor(runs, from <= to)
	if p.err != nil {
		switch p.err {
		case ErrVoidRuns:
			if from == 0 && to == 0 {
				return newRuns()
			}
			return Runs{}, ErrIndexOutOfBounds
		default:
			return Runs{}, p.err
		}
	}
	//if runs.tree == nil || runs.tree.IsEmpty() {
	// if from == 0 && to == 0 {
	// 	return mkEmpty()
	// }
	//return Runs{}, cords.ErrIndexOutOfBounds
	//}
	if total := runs.tree.Summary().length(); to > total {
		return Runs{}, ErrIndexOutOfBounds
	} else if from == to {
		return newRuns()
	}
	_, tail := pipeCall1to2(p, runs.SplitAt, from)
	// _, tail, err := runs.SplitAt(from)
	// if err != nil {
	// 	return Runs{}, err
	// }
	p.runs, _ = pipeCall1to2(p, tail.SplitAt, to-from)
	// section, _, err := tail.SplitAt(to - from)
	// if err != nil {
	// 	return Runs{}, err
	// }
	// return section, nil
	return p.runsOrElse(Runs{}), p.err
}

// DeleteRange removes the run coverage for [from,to) and returns the remaining
// runs, preserving the run invariant.
func (runs Runs) DeleteRange(from, to uint64) (Runs, error) {
	// if from > to {
	// 	return Runs{}, ErrIndexOutOfBounds
	// }
	// if runs.tree == nil || runs.tree.IsEmpty() {
	// 	if from == 0 && to == 0 {
	// 		return runs, nil
	// 	}
	// 	return Runs{}, ErrIndexOutOfBounds
	// }
	p := pipeFor(runs, from <= to)
	if p.err != nil {
		switch p.err {
		case ErrVoidRuns:
			if from == 0 && to == 0 {
				return runs, nil
			}
			return Runs{}, ErrIndexOutOfBounds
		default:
			return Runs{}, p.err
		}
	}
	if total := runs.tree.Summary().length(); to > total {
		return Runs{}, ErrIndexOutOfBounds
	}
	left, tail := pipeCall1to2(p, runs.SplitAt, from)
	// left, tail, err := runs.SplitAt(from)
	// if err != nil {
	// 	return Runs{}, err
	// }
	_, right := pipeCall1to2(p, tail.SplitAt, to-from)
	// _, right, err := tail.SplitAt(to - from)
	// if err != nil {
	// 	return Runs{}, err
	// }
	p.runs = pipeCall1(p, left.Concat, right)
	return p.runsOrElse(Runs{}), p.err
	//return left.Concat(right)
}

// InsertAt inserts n bytes of style run at byte position pos and returns the
// updated runs, preserving the run invariant.
func (runs Runs) InsertAt(pos, n uint64, sty Style) (Runs, error) {
	p := pipeFor(runs)
	//if runs.tree == nil || runs.tree.IsEmpty() {
	if p.err != nil {
		if pos != 0 {
			return Runs{}, ErrIndexOutOfBounds
		}
		if n == 0 {
			return runs, nil
		}
		p.err = nil
		p.runs = pipeCall(p, newRuns)
		// treeRuns, err := newRuns()
		// if err != nil {
		// 	return Runs{}, err
		// }
		run := Run{length: n, style: sty}
		ins1 := func(i int64, r Run) (*btree.Tree[Run, Summary, btree.NO_EXT], error) {
			return p.runs.tree.InsertAt(i, r)
		}
		p.runs.tree = pipeCall2(p, ins1, 0, run)
		// tree, err := treeRuns.tree.InsertAt(0, run)
		// if err != nil {
		// 	return Runs{}, err
		// }
		//return Runs{tree: tree}, nil
		return p.runsOrElse(Runs{}), p.err
	}
	if total := runs.tree.Summary().length(); pos > total {
		return Runs{}, ErrIndexOutOfBounds
	} else if n == 0 {
		return runs, nil
	}
	left, right := pipeCall1to2(p, runs.SplitAt, pos)
	// left, right, err := runs.SplitAt(pos)
	// if err != nil {
	// 	return Runs{}, err
	// }
	inserted := pipeCall(p, newRuns)
	// inserted, err := newRuns()
	// if err != nil {
	// 	return Runs{}, err
	// }
	run := Run{length: n, style: sty}
	ins1 := func(i int, r Run) (*btree.Tree[Run, Summary, btree.NO_EXT], error) {
		return inserted.tree.InsertAt(0, run)
	}
	inserted.tree = pipeCall2(p, ins1, 0, run)
	// inserted.tree, err = inserted.tree.InsertAt(0, run)
	// if err != nil {
	// 	return Runs{}, err
	// }
	p.runs = pipeCall1(p, left.Concat, inserted)
	// out, err := left.Concat(inserted)
	// if err != nil {
	// 	return Runs{}, err
	// }
	p.runs = pipeCall1(p, p.runs.Concat, right)
	//return out.Concat(right)
	return p.runsOrElse(Runs{}), p.err
}

// Style adds a style to (possibly already existing) styles for a given range
// and returns the unified style set (in btree format).
func (runs Runs) Style(textlen uint64, sty Style, from, to uint64) (Runs, error) {
	spn := toSpan(from, to).contained(textlen)
	if spn.void() || textlen == 0 {
		return runs, nil
	}
	if runs.tree == nil || runs.tree.IsEmpty() {
		return initialStyle(textlen, sty, from, to)
	}
	cursor, err := btree.NewCursor(runs.tree, styleDimension{})
	if err != nil {
		return runs, err
	}
	iL, leftRun, leftStart, _, err1 := seekRunForByte(runs.tree, cursor, spn.l)
	iR, rightRun, _, rightEnd, err2 := seekRunForByte(runs.tree, cursor, spn.r-1)
	if err1 != nil || err2 != nil {
		return runs, err1
	}

	repl := make([]Run, 0, 3)
	if spn.l > leftStart {
		repl = append(repl, Run{
			length: spn.l - leftStart,
			style:  leftRun.style,
		})
	}
	repl = append(repl, Run{
		length: spn.len(),
		style:  sty,
	})
	if spn.r < rightEnd {
		repl = append(repl, Run{
			length: rightEnd - spn.r,
			style:  rightRun.style,
		})
	}
	repl = normalizeRunList(repl)
	assert(len(repl) > 0, "impossible: normalized run is void")

	tree, err := runs.tree.DeleteRange(iL, iR-iL+1)
	assert(err == nil, "internal inconsistency")
	tree, err = tree.InsertAt(iL, repl...)
	assert(err == nil, "internal inconsistency")

	leftCanMerge := spn.l == leftStart
	rightCanMerge := spn.r == rightEnd
	insertCount := int64(len(repl))
	leftMerged := false
	if leftCanMerge {
		if tree, leftMerged, err = mergeAdjacentRuns(tree, iL-1); err != nil {
			return runs, err
		}
	}
	if rightCanMerge {
		rightPairLeft := iL + insertCount - 1
		if leftMerged {
			rightPairLeft--
		}
		if tree, _, err = mergeAdjacentRuns(tree, rightPairLeft); err != nil {
			return runs, err
		}
	}
	assert(tree.Summary().length() == textlen, "internal inconsistency")
	return Runs{tree: tree}, nil
}

func normalizeRunList(source []Run) []Run {
	assert(len(source) > 0, "impossible void run list")
	out := make([]Run, 0, len(source))
	for _, run := range source {
		if run.length == 0 {
			continue
		}
		n := len(out)
		if n > 0 && equals(out[n-1].style, run.style) {
			out[n-1].length += run.length
			continue
		}
		out = append(out, run)
	}
	return out
}

func seekRunForByte(
	tree *btree.Tree[Run, Summary, btree.NO_EXT],
	cursor *btree.Cursor[Run, Summary, btree.NO_EXT, uint64],
	pos uint64,
) (index int64, run Run, runStart, runEnd uint64, err error) {
	index, runEnd, err = cursor.Seek(pos + 1)
	assert(err == nil, "internal inconsistency")
	assert(index >= 0 && index < tree.Len(), "run lookup index out of range")
	run, err = tree.At(index)
	assert(err == nil, "internal inconsistency")
	runStart = runEnd - run.length
	assert(pos >= runStart && pos < runEnd, "run lookup mismatch")
	return index, run, runStart, runEnd, nil
}

func mergeAdjacentRuns(
	tree *btree.Tree[Run, Summary, btree.NO_EXT],
	left int64,
) (*btree.Tree[Run, Summary, btree.NO_EXT], bool, error) {
	assert(tree != nil && left >= 0 && left+1 < tree.Len(), "internal inconsistency")
	a, err := tree.At(left)
	assert(err == nil, "internal inconsistency")
	b, err := tree.At(left + 1)
	assert(err == nil, "internal inconsistency")
	if !equals(a.style, b.style) {
		return tree, false, nil
	}
	merged := Run{length: a.length + b.length, style: a.style}
	tree, err = tree.DeleteRange(left, 2)
	assert(err == nil, "internal inconsistency")
	tree, err = tree.InsertAt(left, merged)
	assert(err == nil, "internal inconsistency")
	return tree, true, nil
}
