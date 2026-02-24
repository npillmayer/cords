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
func (t *Tree[I, S, E]) ItemRange(from, to int) iter.Seq2[I, int] {
	var t_, from_, to_ = t, from, to
	return func(yield func(I, int) bool) {
		_ = t_.forEachItemRange(yield, from_, to_)
	}
}

type where struct {
	acc      int // item count left of current leaf
	from, to int
}

func (t *Tree[I, S, E]) forEachItemRange(fn func(I, int) bool, from, to int) error {
	if t == nil || t.root == nil {
		return ErrInvalidConfig
	}
	w := where{acc: 0, from: from, to: to}
	return t.traverseItems(t.root, &w, t.height, fn)
}

// traverseItems traverses the tree in-order, returning items in the range [from,to).
//
// Invariants:
// - height > 0
// - acc >= 0
// - from <= to <= |items|
// - 0 <= i < item.len
// - from <= acc + i < to; otherwise skip or break
func (t *Tree[I, S, E]) traverseItems(n treeNode[I, S, E], w *where, height int,
	fn func(I, int) bool) error {
	//
	assert(n != nil, "tracerseItems called with nil node")
	assert(height > 0, "tracerseItems called with non-positive height")
	if w.acc >= w.to {
		return nil // we are done
	}
	tracer().Debugf("traversing with from=%d, to=%d, acc=%d", w.from, w.to, w.acc)
	if height == 1 {
		leaf := n.(*leafNode[I, S, E])
		assert(w.acc < w.to, "traverseItems: travelled too far")
		tracer().Debugf("visiting leaf %v at height %d", leaf, height)
		if w.acc+int(leaf.n) >= w.from { // leaf contains items in range
			for i := range leaf.n { // iterate over all items of leaf
				if w.acc+int(i) < w.from {
					tracer().Debugf("skipping item %d", i)
					continue // not yet in range
				} else if w.acc+int(i) >= w.to {
					tracer().Debugf("acc = %d, past range", w.acc)
					break // past range
				}
				// now: from <= acc + i < to
				tracer().Debugf("fn(item %d=%v)", i, leaf.items[i])
				fn(leaf.items[i], w.acc+int(i)) // may be `yield(…)`
			}
		}
		w.acc += int(leaf.n) // jump past leaf
		return nil           // possibly go up to next recursion step
	}
	inner := n.(*innerNode[I, S, E])
	tracer().Debugf("visiting inner node %v at height %d", inner, height)
	for _, child := range inner.children {
		childcnt := t.countItems(child)
		if w.acc+childcnt >= w.from { // child contains items in range
			tracer().Debugf("descending with acc=%d", w.acc)
			if err := t.traverseItems(child, w, height-1, fn); err != nil {
				return err
			}
		} else {
			w.acc += childcnt // jump past child
		}
	}
	return nil
}
