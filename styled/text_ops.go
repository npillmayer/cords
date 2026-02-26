package styled

import (
	"github.com/npillmayer/cords"
)

// DeleteRange deletes byte range [from,to) from raw text and synchronizes style
// runs accordingly.
func (t Text) DeleteRange(from, to uint64) (Text, error) {
	p := textPipeFor(t, from <= to)
	if p.err != nil {
		return t, p.err
	}
	// if t == nil {
	// 	return nil, ErrIllegalArguments
	// }
	// if from > to {
	// 	return t, ErrIllegalArguments
	// }
	if to > t.text.Len() {
		return t, ErrIndexOutOfBounds
	} else if from == to {
		return t, nil
	}
	raw, _, err := cords.Cut(t.text, from, to-from)
	if err != nil {
		return t, err
	}
	p_runs := pipeFor(t.runs)
	//updatedRuns := t.runs
	p_runs.runs = t.runs
	if p_runs.err == nil {
		p_runs.runs = pipeCall2(p_runs, t.runs.DeleteRange, from, to)
	}
	// if t.runs.tree != nil && !t.runs.tree.IsEmpty() {
	// 	updatedRuns, err = t.runs.DeleteRange(from, to)
	// 	if err != nil {
	// 		return t, err
	// 	}
	// }
	t.text = raw
	//t.runs = updatedRuns
	t.runs = p_runs.runsOrElse(t.runs)
	return t, nil
}

// InsertAt inserts a cord at byte position pos and synchronizes style runs.
//
// The inserted range receives style sty. If the text has no style runs yet and
// sty is nil, runs stay empty.
func (t Text) InsertAt(pos uint64, insertion cords.Cord, sty Style) (Text, error) {
	p := textPipeFor(t)
	if p.err != nil {
		return t, p.err
	} else if pos > t.text.Len() {
		return t, ErrIndexOutOfBounds
	} else if insertion.Len() == 0 {
		return t, nil
	}
	n := insertion.Len()
	raw, err := cords.Insert(t.text, insertion, pos)
	if err != nil {
		return t, err
	}
	p_runs := pipeFor(t.runs)
	//p_runs.runs = t.runs
	switch p_runs.err {
	case nil:
		p_runs.runs = pipeCall3(p_runs, t.runs.InsertAt, pos, n, sty)
	case ErrVoidRuns:
		p_runs.err = nil
		if sty != nil {
			p_runs.runs, p_runs.err = initialStyle(raw.Len(), sty, pos, pos+n)
		}
	}
	// if p_runs.err == nil {
	// 	p_runs.runs = pipeCall3(p_runs, t.runs.InsertAt, pos, n, sty)
	// } else if p_runs.err == ErrVoidRuns {
	// 	if sty != nil {
	// 		p_runs.err = nil
	// 		p_runs.runs, p_runs.err = initialStyle(raw.Len(), sty, pos, pos+n)
	// 	} else {
	// 		p_runs.err = nil // no runs yet + nil style insertion keeps runs empty
	// 	}
	// }
	if p_runs.err != nil {
		return t, p_runs.err
	}
	t.text = raw
	t.runs = p_runs.runsOrElse(t.runs)
	return t, nil
}

func materializePlainRuns(textlen uint64) (Runs, error) {
	if textlen == 0 {
		return Runs{}, nil
	}
	var runs Runs
	return runs.InsertAt(0, textlen, nil)
}

func ensureRunsForConcat(runs Runs, textlen uint64) (Runs, error) {
	if textlen == 0 {
		return Runs{}, nil
	}
	p := pipeFor(runs)
	if p.err == ErrVoidRuns {
		return materializePlainRuns(textlen)
	}
	if p.err != nil {
		return Runs{}, p.err
	}
	if runs.tree.Summary().length() != textlen {
		return Runs{}, ErrIllegalArguments
	}
	return runs, nil
}

// Concat appends other's content to text and synchronizes style runs.
func (t Text) Concat(other Text) (Text, error) {
	var result Text
	p := textPipeFor(t, !other.text.IsVoid())
	if p.err != nil {
		return t, p.err
	}
	leftLen := t.text.Len()
	rightLen := other.text.Len()
	p.raw = cords.Concat(t.text, other.text)
	// Keep "unstyled" semantics for completely unstyled concat.
	p_runs := pipeFor(t.runs)
	if leftLen+rightLen > 0 {
		p_other := pipeFor(other.runs)
		if p_runs.err != nil && p_other.err != nil {
			t.text = p.raw
			t.runs = Runs{}
			return t, nil
		}
	}
	p_runs.err = nil
	leftRuns := pipeCall2(p_runs, ensureRunsForConcat, t.runs, leftLen)
	// leftRuns, err := ensureRunsForConcat(t.runs, leftLen)
	// if err != nil {
	// 	return t, err
	// }
	rightRuns := pipeCall2(p_runs, ensureRunsForConcat, other.runs, rightLen)
	// rightRuns, err := ensureRunsForConcat(other.runs, rightLen)
	// if err != nil {
	// 	return t, err
	// }
	p_runs.runs = pipeCall1(p_runs, leftRuns.Concat, rightRuns)
	// merged, err := leftRuns.Concat(rightRuns)
	// if err != nil {
	// 	return t, err
	// }
	p.err = p_runs.err
	result.text = p.rawOrElse(t.text)
	result.runs = p_runs.runsOrElse(t.runs)
	// t.text = raw
	// t.runs = merged
	return result, p.err
}

// Section returns a copy of text range [from,to), keeping style runs in sync.
func (t Text) Section(from, to uint64) (Text, error) {
	p := textPipeFor(t, from <= to)
	if p.err != nil {
		return Text{}, p.err
	}
	if to > t.text.Len() {
		return Text{}, ErrIndexOutOfBounds
	}
	raw, err := cords.Substr(t.text, from, to-from)
	if err != nil {
		return Text{}, err
	}
	section := Text{text: raw}
	p_runs := pipeFor(t.runs)
	if p_runs.err == ErrVoidRuns {
		return section, nil // OK: text was unsyled => section is as well
	}
	section.runs = pipeCall2(p_runs, t.runs.Section, from, to)
	return section, p_runs.err
}
