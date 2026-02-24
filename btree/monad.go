package btree

type pipeline[I SummarizedItem[S], S, E any] struct {
	tree    *Tree[I, S, E]
	item    I
	summary S
	err     error
}

func pipeFor[I SummarizedItem[S], S, E any](tree *Tree[I, S, E], conds ...bool) pipeline[I, S, E] {
	p := pipeline[I, S, E]{tree: tree}
	if tree == nil || tree.root == nil {
		p.err = ErrInvalidConfig
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

func (p *pipeline[I, S, E]) call(f func() error) error {
	//
	if p.err != nil {
		return p.err
	}
	p.err = f()
	return p.err
}

func pipeCall[I SummarizedItem[S], S, E, C any](
	p pipeline[I, S, E],
	f func() (C, error),
) (C, error) {
	//
	var c C
	if p.err != nil {
		return c, p.err
	}
	c, p.err = f()
	return c, p.err
}

func pipeCall1[I SummarizedItem[S], S, E, A, C any](
	p pipeline[I, S, E],
	f func(A) (C, error),
	a A,
) (C, error) {
	//
	var c C
	if p.err != nil {
		return c, p.err
	}
	c, p.err = f(a)
	return c, p.err
}

func pipeCall2[I SummarizedItem[S], S, E, A, B, C any](
	p pipeline[I, S, E],
	f func(A, B) (C, error),
	a A, b B,
) (C, error) {
	//
	var c C
	if p.err != nil {
		return c, p.err
	}
	c, p.err = f(a, b)
	return c, p.err
}

func pipeCall3[I SummarizedItem[S], S, E, A, B, C, D any](
	p pipeline[I, S, E],
	f func(A, B, C) (D, error),
	a A, b B, c C,
) (D, error) {
	//
	var d D
	if p.err != nil {
		return d, p.err
	}
	d, p.err = f(a, b, c)
	return d, p.err
}

func (p *pipeline[I, S, E]) itemOrElse(fallback I) I {
	if p.err != nil {
		return fallback
	}
	return p.item
}

func (p *pipeline[I, S, E]) summaryOrElse(fallback S) S {
	if p.err != nil {
		return fallback
	}
	return p.summary
}

func (p *pipeline[I, S, E]) treeOrElse(fallback *Tree[I, S, E]) *Tree[I, S, E] {
	if p.err != nil {
		return fallback
	}
	return p.tree
}
