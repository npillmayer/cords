package btree

// ForEachItem walks leaf items in-order.
//
// Iteration stops early if callback returns false.
func (t *Tree[I, S]) ForEachItem(fn func(item I) bool) {
	if t == nil || t.root == nil || fn == nil {
		return
	}
	t.forEachItemNode(t.root, fn)
}

func (t *Tree[I, S]) forEachItemNode(n treeNode[I, S], fn func(item I) bool) bool {
	assert(n != nil, "forEachItemNode called with nil node")
	if n.isLeaf() {
		leaf := n.(*leafNode[I, S])
		for _, item := range leaf.items {
			if !fn(item) {
				return false
			}
		}
		return true
	}
	inner := n.(*innerNode[I, S])
	for _, child := range inner.children {
		if !t.forEachItemNode(child, fn) {
			return false
		}
	}
	return true
}
