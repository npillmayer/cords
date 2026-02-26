package styled

import (
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/cordext"
)

// We create a couple of monadic helpers to make error-handling easier on the eye.
// The boilerplate code in this file is a bit cumbersome, but application of the
// pipeline functions will read rather easily.
//
// Use in this manner:
//
//	p := pipeFor(myruns, myArg != 0)
//	thing1 := pipeCall2(p, someFunc, param1, param2)
//	p.item := pipeCall1(p, anotherFunc, param)
//	item := p.itemOrElse(defaultVal)
//	return item, p.err
//
// The pipe... calls are wildly type-parameterized, but using them should be
// straightforward.
type pipeline struct {
	runs    Runs
	run     Run
	summary Summary
	err     error
}

func pipeFor(runs Runs, conds ...bool) pipeline {
	p := pipeline{runs: runs}
	if runs.tree == nil || runs.tree.IsEmpty() {
		p.err = ErrVoidRuns
		return p
	}
	for _, cond := range conds {
		if !cond {
			p.err = ErrIllegalArguments
		}
	}
	return p
}

// We have to circumvent that Go does not allow generic member functions

func (p *pipeline) call(f func() error) error {
	//
	if p.err != nil {
		return p.err
	}
	p.err = f()
	return p.err
}

func pipeCall[C any](
	p pipeline,
	f func() (C, error),
) C {
	//
	var c C
	if p.err != nil {
		return c
	}
	c, p.err = f()
	return c
}

func pipeCall1[A, C any](
	p pipeline,
	f func(A) (C, error),
	a A,
) C {
	//
	var c C
	if p.err != nil {
		return c
	}
	c, p.err = f(a)
	return c
}

func pipeCall1to2[A, C, D any](
	p pipeline,
	f func(A) (C, D, error),
	a A,
) (C, D) {
	//
	var c C
	var d D
	if p.err != nil {
		return c, d
	}
	c, d, p.err = f(a)
	return c, d
}

func pipeCall2[A, B, C any](
	p pipeline,
	f func(A, B) (C, error),
	a A, b B,
) C {
	//
	var c C
	if p.err != nil {
		return c
	}
	c, p.err = f(a, b)
	return c
}

func pipeCall2to2[A, B, C, D any](
	p pipeline,
	f func(A, B) (C, D, error),
	a A, b B,
) (C, D) {
	//
	var c C
	var d D
	if p.err != nil {
		return c, d
	}
	c, d, p.err = f(a, b)
	return c, d
}

func pipeCall2a[A, B, D any](
	p pipeline,
	f func(A, B, B) (D, error),
	a A, b B, c B,
) D {
	//
	var d D
	if p.err != nil {
		return d
	}
	d, p.err = f(a, b, c)
	return d
}

func pipeCall3[A, B, C, D any](
	p pipeline,
	f func(A, B, C) (D, error),
	a A, b B, c C,
) D {
	//
	var d D
	if p.err != nil {
		return d
	}
	d, p.err = f(a, b, c)
	return d
}

func (p *pipeline) runOrElse(fallback Run) Run {
	if p.err != nil {
		return fallback
	}
	return p.run
}

func (p *pipeline) summaryOrElse(fallback Summary) Summary {
	if p.err != nil {
		return fallback
	}
	return p.summary
}

func (p *pipeline) runsOrElse(fallback Runs) Runs {
	if p.err != nil {
		return fallback
	}
	return p.runs
}

// ---------------------------------------------------------------------------

type textPipeline struct {
	text *Text
	raw  cordext.CordEx[btree.NO_EXT]
	err  error
}

func textPipeFor(text Text, conds ...bool) textPipeline {
	p := textPipeline{text: &text}
	if text.text.IsVoid() {
		p.err = ErrVoidText
		return p
	}
	for _, cond := range conds {
		if !cond {
			p.err = ErrIllegalArguments
		}
	}
	return p
}

func (p *textPipeline) rawOrElse(fallback cordext.CordEx[btree.NO_EXT]) cordext.CordEx[btree.NO_EXT] {
	if p.err != nil {
		return fallback
	}
	return p.raw
}
