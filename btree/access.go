package btree

// At returns the leaf item at item index.
func (t *Tree[I, S, E]) At(index int) (I, error) {
	var zero I
	if t == nil || t.root == nil {
		return zero, ErrIndexOutOfBounds
	}
	if index < 0 || index >= t.Len() {
		return zero, ErrIndexOutOfBounds
	}
	return t.atNode(t.root, t.height, index)
}

func (t *Tree[I, S, E]) atNode(n treeNode[I, S, E], height int, index int) (I, error) {
	var zero I
	assert(n != nil, "atNode called with nil node")
	assert(height > 0, "atNode called with non-positive height")
	if height == 1 {
		leaf := n.(*leafNode[I, S, E])
		if index < 0 || index >= len(leaf.items) {
			return zero, ErrIndexOutOfBounds
		}
		return leaf.items[index], nil
	}
	inner := n.(*innerNode[I, S, E])
	remaining := index
	for _, child := range inner.children {
		childItems := t.countItems(child)
		if remaining < childItems {
			return t.atNode(child, height-1, remaining)
		}
		remaining -= childItems
	}
	assert(false, "atNode index routing exceeded subtree size")
	return zero, ErrIndexOutOfBounds
}

// PrefixSummary returns the aggregated summary for items [0,itemIndex).
//
// itemIndex may be equal to Len(), in which case the full tree summary is
// returned. itemIndex must not be negative.
func (t *Tree[I, S, E]) PrefixSummary(itemIndex int) (S, error) {
	zero := t.cfg.Monoid.Zero()
	if t == nil || t.root == nil {
		if itemIndex == 0 {
			return zero, nil
		}
		return zero, ErrIndexOutOfBounds
	}
	if itemIndex < 0 || itemIndex > t.Len() {
		return zero, ErrIndexOutOfBounds
	}
	if itemIndex == 0 {
		return zero, nil
	}
	return t.prefixSummaryNode(t.root, t.height, itemIndex, zero)
}

// PrefixExt returns the aggregated extension value for items [0,itemIndex).
//
// itemIndex may be equal to Len(), in which case the full extension value is
// returned. itemIndex must not be negative.
func (t *Tree[I, S, E]) PrefixExt(itemIndex int) (E, error) {
	var zero E
	if t == nil {
		return zero, ErrInvalidConfig
	}
	if t.cfg.Extension == nil {
		return zero, ErrInvalidConfig
	}
	if t.root == nil {
		if itemIndex == 0 {
			return t.cfg.Extension.Zero(), nil
		}
		return zero, ErrIndexOutOfBounds
	}
	if itemIndex < 0 || itemIndex > t.Len() {
		return zero, ErrIndexOutOfBounds
	}
	if itemIndex == 0 {
		return t.cfg.Extension.Zero(), nil
	}
	return t.prefixExtNode(t.root, t.height, itemIndex, t.cfg.Extension.Zero())
}

func (t *Tree[I, S, E]) prefixSummaryNode(n treeNode[I, S, E], height int, remaining int, acc S) (S, error) {
	assert(n != nil, "prefixSummaryNode called with nil node")
	assert(height > 0, "prefixSummaryNode called with non-positive height")
	assert(remaining >= 0, "prefixSummaryNode called with negative remaining")

	if remaining == 0 {
		return acc, nil
	}
	if height == 1 {
		leaf := n.(*leafNode[I, S, E])
		if remaining > len(leaf.items) {
			var zero S
			return zero, ErrIndexOutOfBounds
		}
		sum := acc
		for i := 0; i < remaining; i++ {
			sum = t.cfg.Monoid.Add(sum, leaf.items[i].Summary())
		}
		return sum, nil
	}

	inner := n.(*innerNode[I, S, E])
	sum := acc
	rem := remaining
	for _, child := range inner.children {
		childItems := t.countItems(child)
		if rem >= childItems {
			sum = t.cfg.Monoid.Add(sum, child.Summary())
			rem -= childItems
			if rem == 0 {
				return sum, nil
			}
			continue
		}
		return t.prefixSummaryNode(child, height-1, rem, sum)
	}
	assert(false, "prefixSummaryNode routing exceeded subtree size")
	var zero S
	return zero, ErrIndexOutOfBounds
}

func (t *Tree[I, S, E]) prefixExtNode(n treeNode[I, S, E], height int, remaining int, acc E) (E, error) {
	assert(n != nil, "prefixExtNode called with nil node")
	assert(height > 0, "prefixExtNode called with non-positive height")
	assert(remaining >= 0, "prefixExtNode called with negative remaining")

	if remaining == 0 {
		return acc, nil
	}
	if height == 1 {
		leaf := n.(*leafNode[I, S, E])
		if remaining > len(leaf.items) {
			var zero E
			return zero, ErrIndexOutOfBounds
		}
		sum := acc
		for i := 0; i < remaining; i++ {
			item := leaf.items[i]
			right := t.cfg.Extension.FromItem(item, item.Summary())
			sum = t.cfg.Extension.Add(sum, right)
		}
		return sum, nil
	}

	inner := n.(*innerNode[I, S, E])
	sum := acc
	rem := remaining
	for _, child := range inner.children {
		childItems := t.countItems(child)
		if rem >= childItems {
			sum = t.cfg.Extension.Add(sum, child.Ext())
			rem -= childItems
			if rem == 0 {
				return sum, nil
			}
			continue
		}
		return t.prefixExtNode(child, height-1, rem, sum)
	}
	assert(false, "prefixExtNode routing exceeded subtree size")
	var zero E
	return zero, ErrIndexOutOfBounds
}
