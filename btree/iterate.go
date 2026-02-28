package btree

import (
	"iter"
)

// ForEachItem walks leaf items in-order.
//
// Iteration stops early if callback returns false.
func (t *Tree[I, S, E]) ForEachItem(fn func(item I) bool) {
	if t == nil || t.root == nil || fn == nil {
		return
	}
	t.forEachItemNode(t.root, fn)
}

func (t *Tree[I, S, E]) forEachItemNode(n treeNode[I, S, E], fn func(item I) bool) bool {
	assert(n != nil, "forEachItemNode called with nil node")
	if n.isLeaf() {
		leaf := n.(*leafNode[I, S, E])
		for _, item := range leaf.items {
			if !fn(item) {
				return false
			}
		}
		return true
	}
	inner := n.(*innerNode[I, S, E])
	for _, child := range inner.children {
		if !t.forEachItemNode(child, fn) {
			return false
		}
	}
	return true
}

// ItemRange returns a range iterator over items in [from,to).
//
// The yielded pair is (item, absoluteItemIndex). ItemRange delegates to the
// internal traversal helper and currently suppresses traversal errors in the
// iterator closure.
func (t *Tree[I, S, E]) ItemRange(from, to int64) iter.Seq2[int64, I] {
	var t_, from_, to_ = t, from, to
	return func(yield func(int64, I) bool) {
		_, _ = t_.forEachItemRange(yield, from_, to_)
	}
}

type where[I any] struct {
	acc      int64               // item count to the left of current leaf, variable
	from, to int64               // const
	fn       func(int64, I) bool // const
}

func (t *Tree[I, S, E]) forEachItemRange(fn func(int64, I) bool, from, to int64) (int64, error) {
	w := where[I]{acc: 0, from: from, to: to, fn: fn}
	p := pipeFor(t, fn != nil, from < to)
	acc := pipeCall3(p, t.traverseItems, t.root, &w, t.height)
	return acc, p.err
}

// traverseItems traverses the tree in-order, returning items in the range [from,to).
//
// Invariants:
// - height > 0
// - acc >= 0
// - from <= to <= |items|
// - 0 <= i < item.len
// - from <= acc + i < to; otherwise skip or break
func (t *Tree[I, S, E]) traverseItems(n treeNode[I, S, E], w *where[I], height int) (
	int64, error) {
	//
	assert(n != nil, "traverseItems called with nil node")
	assert(height > 0, "traverseItems called with non-positive height")
	if w.acc >= w.to {
		return w.acc, nil // we are done
	}
	if height == 1 { // we are in a leaf node
		leaf := n.(*leafNode[I, S, E])
		assert(w.acc < w.to, "traverseItems: travelled too far")
		if w.acc+int64(leaf.n) >= w.from { // leaf contains items in range
			for i := range leaf.n { // iterate over all items of leaf
				if w.acc+int64(i) < w.from {
					continue // not yet in range
				} else if w.acc+int64(i) >= w.to {
					break // past range
				}
				// now: from <= acc + i < to
				w.fn(w.acc+int64(i), leaf.items[i]) // may be `yield(…)`
			}
		}
		w.acc += int64(leaf.n) // jump past leaf
		return w.acc, nil      // possibly go up to next recursion step
	}
	inner := n.(*innerNode[I, S, E])
	for _, child := range inner.children {
		// todo remove
		//itemcnt := t.countItems(child)
		itemcnt := child.Weight()
		if w.acc+itemcnt >= w.from { // child contains items in range
			if n, err := t.traverseItems(child, w, height-1); err != nil {
				return n, err
			}
		} else {
			w.acc += itemcnt // jump past child
		}
	}
	return w.acc, nil
}
