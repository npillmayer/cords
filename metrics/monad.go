package metrics

type pipeline struct {
	cord plainCordType
	// run     Run
	// summary Summary
	err error
}

func pipeFor(cord plainCordType, conds ...bool) pipeline {
	p := pipeline{cord: cord}
	if cord.Tree() == nil || cord.Tree().IsEmpty() {
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

// func (p *pipeline) runOrElse(fallback Run) Run {
// 	if p.err != nil {
// 		return fallback
// 	}
// 	return p.run
// }
