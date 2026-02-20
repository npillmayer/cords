package btree

// cloneNode clones a node for path-copy updates.
func (t *Tree[I, S, E]) cloneNode(n treeNode[I, S, E]) treeNode[I, S, E] {
	if n == nil {
		return nil
	}
	switch n := n.(type) {
	case *leafNode[I, S, E]:
		return t.cloneLeaf(n)
	case *innerNode[I, S, E]:
		return t.cloneInner(n)
	default:
		panic("unknown tree node type")
	}
}

func (t *Tree[I, S, E]) cloneLeaf(leaf *leafNode[I, S, E]) *leafNode[I, S, E] {
	assert(leaf != nil, "cloneLeaf called with nil leaf")
	cloned := &leafNode[I, S, E]{
		summary: leaf.summary,
		ext:     leaf.ext,
		n:       leaf.n,
	}
	copy(cloned.itemStore[:int(cloned.n)], leaf.itemStore[:int(leaf.n)])
	cloned.items = cloned.itemStore[:int(cloned.n)]
	return cloned
}

// cloneInner copies an internal node including its fixed backing storage.
//
// The returned node is independent and safe to mutate on the current path-copy
// operation.
func (t *Tree[I, S, E]) cloneInner(inner *innerNode[I, S, E]) *innerNode[I, S, E] {
	assert(inner != nil, "cloneInner called with nil inner node")
	cloned := &innerNode[I, S, E]{
		summary: inner.summary,
		ext:     inner.ext,
		n:       inner.n,
	}
	copy(cloned.childStore[:int(cloned.n)], inner.childStore[:int(inner.n)])
	cloned.children = cloned.childStore[:int(cloned.n)]
	return cloned
}

// recomputeNodeSummary recalculates summary from immediate children/items.
//
// This is used after local structural edits. It does not recurse because child
// summaries are already maintained by lower-level edits.
func (t *Tree[I, S, E]) recomputeNodeSummary(n treeNode[I, S, E]) {
	assert(n != nil, "recomputeNodeSummary called with nil node")
	switch n := n.(type) {
	case *leafNode[I, S, E]:
		t.recomputeLeafSummary(n)
	case *innerNode[I, S, E]:
		t.recomputeInnerSummary(n)
	default:
		panic("unknown tree node type")
	}
}

// recomputeLeafSummary rebuilds a leaf summary (and extension, if configured)
// from item summaries.
func (t *Tree[I, S, E]) recomputeLeafSummary(leaf *leafNode[I, S, E]) {
	assert(leaf != nil, "recomputeLeafSummary called with nil leaf")
	leaf.summary = t.cfg.Monoid.Zero()
	var zeroE E
	leaf.ext = zeroE
	if t.cfg.Extension != nil {
		leaf.ext = t.cfg.Extension.Zero()
	}
	for _, item := range leaf.items {
		leaf.summary = t.cfg.Monoid.Add(leaf.summary, item.Summary())
		if t.cfg.Extension != nil {
			right := t.cfg.Extension.FromItem(item, item.Summary())
			leaf.ext = t.cfg.Extension.Add(leaf.ext, right)
		}
	}
}

// recomputeInnerSummary rebuilds an internal summary (and extension, if
// configured) by folding child summaries.
func (t *Tree[I, S, E]) recomputeInnerSummary(inner *innerNode[I, S, E]) {
	assert(inner != nil, "recomputeInnerSummary called with nil inner node")
	inner.summary = t.cfg.Monoid.Zero()
	var zeroE E
	inner.ext = zeroE
	if t.cfg.Extension != nil {
		inner.ext = t.cfg.Extension.Zero()
	}
	for _, child := range inner.children {
		if child != nil {
			inner.summary = t.cfg.Monoid.Add(inner.summary, child.Summary())
			if t.cfg.Extension != nil {
				inner.ext = t.cfg.Extension.Add(inner.ext, child.Ext())
			}
		}
	}
}

// insertAt inserts values into a slice at idx and returns a new slice.
func insertAt[T any](src []T, idx int, values ...T) []T {
	assert(idx >= 0 && idx <= len(src), "insertAt index out of range")
	if len(values) == 0 {
		return append([]T(nil), src...)
	}
	out := make([]T, 0, len(src)+len(values))
	out = append(out, src[:idx]...)
	out = append(out, values...)
	out = append(out, src[idx:]...)
	return out
}

// removeRange removes the half-open interval [from,to) from a slice.
func removeRange[T any](src []T, from, to int) []T {
	assert(from >= 0 && from <= to && to <= len(src), "removeRange bounds invalid")
	out := make([]T, 0, len(src)-(to-from))
	out = append(out, src[:from]...)
	out = append(out, src[to:]...)
	return out
}

func (t *Tree[I, S, E]) insertChildAt(inner *innerNode[I, S, E], idx int, child treeNode[I, S, E]) {
	assert(inner != nil, "insertChildAt called with nil inner node")
	assert(idx >= 0 && idx <= len(inner.children), "insertChildAt index out of range")
	n := len(inner.children)
	assert(n < len(inner.childStore), "insertChildAt exceeds fixed child capacity")
	if idx < n {
		copy(inner.childStore[idx+1:n+1], inner.childStore[idx:n])
	}
	inner.childStore[idx] = child
	inner.n = uint8(n + 1)
	inner.children = inner.childStore[:n+1]
	t.recomputeInnerSummary(inner)
}

// removeChildAt removes one child pointer from an internal node at idx.
//
// It compacts the fixed backing store and refreshes summary.
func (t *Tree[I, S, E]) removeChildAt(inner *innerNode[I, S, E], idx int) {
	assert(inner != nil, "removeChildAt called with nil inner node")
	assert(idx >= 0 && idx < len(inner.children), "removeChildAt index out of range")
	n := len(inner.children)
	if idx < n-1 {
		copy(inner.childStore[idx:n-1], inner.childStore[idx+1:n])
	}
	var zeroNode treeNode[I, S, E]
	inner.childStore[n-1] = zeroNode
	inner.n = uint8(n - 1)
	inner.children = inner.childStore[:n-1]
	t.recomputeInnerSummary(inner)
}

// insertLeafItemsAt inserts one or more items at leaf-local idx.
//
// The leaf may temporarily exceed normal occupancy (up to overflow storage),
// which is resolved by higher-level split logic.
func (t *Tree[I, S, E]) insertLeafItemsAt(leaf *leafNode[I, S, E], idx int, values ...I) {
	assert(leaf != nil, "insertLeafItemsAt called with nil leaf")
	assert(idx >= 0 && idx <= len(leaf.items), "insertLeafItemsAt index out of range")
	if len(values) == 0 {
		return
	}
	n := len(leaf.items)
	k := len(values)
	assert(n+k <= len(leaf.itemStore), "insertLeafItemsAt exceeds fixed leaf capacity")
	if idx < n {
		copy(leaf.itemStore[idx+k:n+k], leaf.itemStore[idx:n])
	}
	copy(leaf.itemStore[idx:idx+k], values)
	leaf.n = uint8(n + k)
	leaf.items = leaf.itemStore[:n+k]
}

// removeLeafItemsRange removes half-open interval [from,to) from a leaf.
func (t *Tree[I, S, E]) removeLeafItemsRange(leaf *leafNode[I, S, E], from, to int) {
	assert(leaf != nil, "removeLeafItemsRange called with nil leaf")
	assert(from >= 0 && from <= to && to <= len(leaf.items), "removeLeafItemsRange bounds invalid")
	if from == to {
		return
	}
	n := len(leaf.items)
	k := to - from
	if to < n {
		copy(leaf.itemStore[from:n-k], leaf.itemStore[to:n])
	}
	var zero I
	for i := n - k; i < n; i++ {
		leaf.itemStore[i] = zero
	}
	leaf.n = uint8(n - k)
	leaf.items = leaf.itemStore[:n-k]
}

// leafOverflow reports whether leaf exceeds the allowed non-overflow occupancy.
func (t *Tree[I, S, E]) leafOverflow(leaf *leafNode[I, S, E]) bool {
	return leaf != nil && len(leaf.items) > MaxLeafItems
}

// leafUnderflow reports whether a non-root leaf violates minimum occupancy.
func (t *Tree[I, S, E]) leafUnderflow(leaf *leafNode[I, S, E], isRoot bool) bool {
	assert(leaf != nil, "leafUnderflow called with nil leaf")
	if isRoot {
		return false
	}
	return len(leaf.items) < Base
}

// innerOverflow reports whether internal node exceeds maximum children.
func (t *Tree[I, S, E]) innerOverflow(inner *innerNode[I, S, E]) bool {
	return inner != nil && len(inner.children) > MaxChildren
}

// innerUnderflow reports whether a non-root internal node is below min fill.
func (t *Tree[I, S, E]) innerUnderflow(inner *innerNode[I, S, E], isRoot bool) bool {
	assert(inner != nil, "innerUnderflow called with nil inner node")
	if isRoot {
		return false
	}
	return len(inner.children) < Base
}

// insertIntoLeafLocal inserts items at a local leaf offset.
//
// It returns the updated (left) leaf and optionally a promoted right sibling if
// a split occurred.
func (t *Tree[I, S, E]) insertIntoLeafLocal(leaf *leafNode[I, S, E], index int, items ...I) (*leafNode[I, S, E], *leafNode[I, S, E], error) {
	assert(leaf != nil, "insertIntoLeafLocal called with nil leaf")
	assert(index >= 0 && index <= len(leaf.items), "insertIntoLeafLocal index out of range")
	if len(items) == 0 {
		return t.cloneLeaf(leaf), nil, nil
	}
	cloned := t.cloneLeaf(leaf)
	t.insertLeafItemsAt(cloned, index, items...)
	t.recomputeLeafSummary(cloned)
	if !t.leafOverflow(cloned) {
		return cloned, nil, nil
	}
	left, right := t.splitLeaf(cloned)
	return left, right, nil
}

// splitLeaf splits an overflowing leaf into two siblings.
//
// The split is midpoint-based and guarantees both outputs satisfy non-root
// minimum occupancy.
func (t *Tree[I, S, E]) splitLeaf(leaf *leafNode[I, S, E]) (*leafNode[I, S, E], *leafNode[I, S, E]) {
	assert(leaf != nil, "splitLeaf called with nil leaf")
	n := len(leaf.items)
	maxItems := MaxLeafItems
	if n <= maxItems {
		return t.cloneLeaf(leaf), nil
	}
	assert(n <= 2*maxItems, "splitLeaf requires more than one promoted sibling")
	mid := n / 2
	left := t.makeLeaf(leaf.items[:mid])
	right := t.makeLeaf(leaf.items[mid:])
	assert(len(left.items) >= Base && len(right.items) >= Base,
		"splitLeaf violates leaf occupancy bounds")
	return left, right
}
