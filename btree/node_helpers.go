package btree

// makeLeaf materializes a new leaf backed by fixed inline storage and computes
// its summary.
func (t *Tree[I, S]) makeLeaf(items []I) *leafNode[I, S] {
	leaf := &leafNode[I, S]{}
	assert(len(items) <= len(leaf.itemStore), "makeLeaf exceeds fixed leaf capacity")
	copy(leaf.itemStore[:], items)
	leaf.n = uint8(len(items))
	leaf.items = leaf.itemStore[:len(items)]
	leaf.summary = t.cfg.Monoid.Zero()
	for _, item := range leaf.items {
		leaf.summary = t.cfg.Monoid.Add(leaf.summary, item.Summary())
	}
	return leaf
}

// makeInternal materializes a new internal node backed by fixed inline storage
// and computes its summary from child summaries.
func (t *Tree[I, S]) makeInternal(children ...treeNode[I, S]) *innerNode[I, S] {
	inner := &innerNode[I, S]{}
	assert(len(children) <= len(inner.childStore), "makeInternal exceeds fixed node capacity")
	copy(inner.childStore[:], children)
	inner.n = uint8(len(children))
	inner.children = inner.childStore[:len(children)]
	inner.summary = t.cfg.Monoid.Zero()
	for _, child := range inner.children {
		inner.summary = t.cfg.Monoid.Add(inner.summary, child.Summary())
	}
	return inner
}
