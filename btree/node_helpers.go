//go:build !btree_fixed

package btree

func (t *Tree[I, S]) makeLeaf(items []I) *leafNode[I, S] {
	leaf := &leafNode[I, S]{
		items: append([]I(nil), items...),
	}
	leaf.summary = t.cfg.Monoid.Zero()
	for _, item := range leaf.items {
		leaf.summary = t.cfg.Monoid.Add(leaf.summary, item.Summary())
	}
	return leaf
}

func (t *Tree[I, S]) makeInternal(children ...treeNode[I, S]) *innerNode[I, S] {
	inner := &innerNode[I, S]{
		children: append([]treeNode[I, S](nil), children...),
	}
	inner.summary = t.cfg.Monoid.Zero()
	for _, child := range children {
		inner.summary = t.cfg.Monoid.Add(inner.summary, child.Summary())
	}
	return inner
}
