package btree

// makeLeaf materializes a new leaf backed by fixed inline storage and computes
// its summary.
func (t *Tree[I, S, E]) makeLeaf(items []I) *leafNode[I, S, E] {
	leaf := &leafNode[I, S, E]{}
	assert(len(items) <= len(leaf.itemStore), "makeLeaf exceeds fixed leaf capacity")
	copy(leaf.itemStore[:], items)
	leaf.n = uint8(len(items))
	leaf.items = leaf.itemStore[:len(items)]
	leaf.summary = t.cfg.Monoid.Zero()
	if t.cfg.Extension != nil {
		leaf.ext = t.cfg.Extension.Zero()
		for _, item := range leaf.items {
			right := t.cfg.Extension.FromItem(item, item.Summary())
			leaf.ext = t.cfg.Extension.Add(leaf.ext, right)
		}
	}
	for _, item := range leaf.items {
		leaf.summary = t.cfg.Monoid.Add(leaf.summary, item.Summary())
	}
	return leaf
}

// makeInternal materializes a new internal node backed by fixed inline storage
// and computes its summary from child summaries.
func (t *Tree[I, S, E]) makeInternal(children ...treeNode[I, S, E]) *innerNode[I, S, E] {
	inner := &innerNode[I, S, E]{}
	assert(len(children) <= len(inner.childStore), "makeInternal exceeds fixed node capacity")
	copy(inner.childStore[:], children)
	inner.n = uint8(len(children))
	inner.children = inner.childStore[:len(children)]
	inner.summary = t.cfg.Monoid.Zero()
	if t.cfg.Extension != nil {
		inner.ext = t.cfg.Extension.Zero()
		for _, child := range inner.children {
			inner.ext = t.cfg.Extension.Add(inner.ext, child.Ext())
		}
	}
	for _, child := range inner.children {
		inner.summary = t.cfg.Monoid.Add(inner.summary, child.Summary())
	}
	return inner
}
