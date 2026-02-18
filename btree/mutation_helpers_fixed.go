//go:build btree_fixed

package btree

import "fmt"

// cloneNode clones a node for path-copy updates.
func (t *Tree[I, S]) cloneNode(n treeNode[I, S]) treeNode[I, S] {
	if n == nil {
		return nil
	}
	switch n := n.(type) {
	case *leafNode[I, S]:
		return t.cloneLeaf(n)
	case *innerNode[I, S]:
		return t.cloneInner(n)
	default:
		panic("unknown tree node type")
	}
}

func (t *Tree[I, S]) cloneLeaf(leaf *leafNode[I, S]) *leafNode[I, S] {
	if leaf == nil {
		return nil
	}
	cloned := &leafNode[I, S]{
		summary: leaf.summary,
		n:       leaf.n,
	}
	copy(cloned.itemStore[:int(cloned.n)], leaf.itemStore[:int(leaf.n)])
	cloned.items = cloned.itemStore[:int(cloned.n)]
	return cloned
}

func (t *Tree[I, S]) cloneInner(inner *innerNode[I, S]) *innerNode[I, S] {
	if inner == nil {
		return nil
	}
	cloned := &innerNode[I, S]{
		summary: inner.summary,
		n:       inner.n,
	}
	copy(cloned.childStore[:int(cloned.n)], inner.childStore[:int(inner.n)])
	cloned.children = cloned.childStore[:int(cloned.n)]
	return cloned
}

func (t *Tree[I, S]) recomputeNodeSummary(n treeNode[I, S]) error {
	if n == nil {
		return fmt.Errorf("%w: nil node", ErrInvalidConfig)
	}
	switch n := n.(type) {
	case *leafNode[I, S]:
		t.recomputeLeafSummary(n)
	case *innerNode[I, S]:
		t.recomputeInnerSummary(n)
	default:
		panic("unknown tree node type")
	}
	return nil
}

func (t *Tree[I, S]) recomputeLeafSummary(leaf *leafNode[I, S]) {
	assert(leaf != nil, "recomputeLeafSummary called with nil leaf")
	leaf.summary = t.cfg.Monoid.Zero()
	for _, item := range leaf.items {
		leaf.summary = t.cfg.Monoid.Add(leaf.summary, item.Summary())
	}
}

func (t *Tree[I, S]) recomputeInnerSummary(inner *innerNode[I, S]) {
	assert(inner != nil, "recomputeInnerSummary called with nil inner node")
	inner.summary = t.cfg.Monoid.Zero()
	for _, child := range inner.children {
		if child != nil {
			inner.summary = t.cfg.Monoid.Add(inner.summary, child.Summary())
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

func (t *Tree[I, S]) insertChildAt(inner *innerNode[I, S], idx int, child treeNode[I, S]) {
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

func (t *Tree[I, S]) removeChildAt(inner *innerNode[I, S], idx int) {
	assert(inner != nil, "removeChildAt called with nil inner node")
	assert(idx >= 0 && idx < len(inner.children), "removeChildAt index out of range")
	n := len(inner.children)
	if idx < n-1 {
		copy(inner.childStore[idx:n-1], inner.childStore[idx+1:n])
	}
	var zeroNode treeNode[I, S]
	inner.childStore[n-1] = zeroNode
	inner.n = uint8(n - 1)
	inner.children = inner.childStore[:n-1]
	t.recomputeInnerSummary(inner)
}

func (t *Tree[I, S]) insertLeafItemsAt(leaf *leafNode[I, S], idx int, values ...I) error {
	assert(leaf != nil, "insertLeafItemsAt called with nil leaf")
	assert(idx >= 0 && idx <= len(leaf.items), "insertLeafItemsAt index out of range")
	if len(values) == 0 {
		return nil
	}
	n := len(leaf.items)
	k := len(values)
	if n+k > len(leaf.itemStore) {
		return fmt.Errorf("%w: fixed leaf capacity exceeded", ErrUnimplemented)
	}
	if idx < n {
		copy(leaf.itemStore[idx+k:n+k], leaf.itemStore[idx:n])
	}
	copy(leaf.itemStore[idx:idx+k], values)
	leaf.n = uint8(n + k)
	leaf.items = leaf.itemStore[:n+k]
	return nil
}

func (t *Tree[I, S]) removeLeafItemsRange(leaf *leafNode[I, S], from, to int) {
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

func (t *Tree[I, S]) maxLeafItems() int {
	if t.cfg.Degree < fixedMaxLeafItems {
		return t.cfg.Degree
	}
	return fixedMaxLeafItems
}

func (t *Tree[I, S]) minLeafItems() int {
	return t.cfg.MinFill
}

func (t *Tree[I, S]) maxChildren() int {
	if t.cfg.Degree < fixedMaxChildren {
		return t.cfg.Degree
	}
	return fixedMaxChildren
}

func (t *Tree[I, S]) minChildren() int {
	return t.cfg.MinFill
}

func (t *Tree[I, S]) leafOverflow(leaf *leafNode[I, S]) bool {
	return leaf != nil && len(leaf.items) > t.maxLeafItems()
}

func (t *Tree[I, S]) leafUnderflow(leaf *leafNode[I, S], isRoot bool) bool {
	if leaf == nil {
		return false
	}
	if isRoot {
		return false
	}
	return len(leaf.items) < t.minLeafItems()
}

func (t *Tree[I, S]) innerOverflow(inner *innerNode[I, S]) bool {
	return inner != nil && len(inner.children) > t.maxChildren()
}

func (t *Tree[I, S]) innerUnderflow(inner *innerNode[I, S], isRoot bool) bool {
	if inner == nil {
		return false
	}
	if isRoot {
		return false
	}
	return len(inner.children) < t.minChildren()
}

// insertIntoLeafLocal inserts items at a local leaf offset.
//
// It returns the updated (left) leaf and optionally a promoted right sibling if
// a split occurred.
func (t *Tree[I, S]) insertIntoLeafLocal(leaf *leafNode[I, S], index int, items ...I) (*leafNode[I, S], *leafNode[I, S], error) {
	if leaf == nil {
		return nil, nil, fmt.Errorf("%w: nil leaf", ErrInvalidConfig)
	}
	if index < 0 || index > len(leaf.items) {
		return nil, nil, ErrIndexOutOfBounds
	}
	if len(items) == 0 {
		return t.cloneLeaf(leaf), nil, nil
	}
	cloned := t.cloneLeaf(leaf)
	if err := t.insertLeafItemsAt(cloned, index, items...); err != nil {
		return nil, nil, err
	}
	t.recomputeLeafSummary(cloned)
	if !t.leafOverflow(cloned) {
		return cloned, nil, nil
	}
	left, right, err := t.splitLeaf(cloned)
	return left, right, err
}

// splitLeaf splits an overflowing leaf into two siblings.
func (t *Tree[I, S]) splitLeaf(leaf *leafNode[I, S]) (*leafNode[I, S], *leafNode[I, S], error) {
	if leaf == nil {
		return nil, nil, fmt.Errorf("%w: nil leaf", ErrInvalidConfig)
	}
	n := len(leaf.items)
	maxItems := t.maxLeafItems()
	if n <= maxItems {
		return t.cloneLeaf(leaf), nil, nil
	}
	if n > 2*maxItems {
		return nil, nil, fmt.Errorf("%w: leaf split requires more than one sibling", ErrUnimplemented)
	}
	mid := n / 2
	left := t.makeLeaf(leaf.items[:mid])
	right := t.makeLeaf(leaf.items[mid:])
	if len(left.items) < t.minLeafItems() || len(right.items) < t.minLeafItems() {
		return nil, nil, fmt.Errorf("%w: split violates leaf occupancy bounds", ErrInvalidConfig)
	}
	return left, right, nil
}
