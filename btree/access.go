package btree

// At returns the leaf item at item index.
func (t *Tree[I, S]) At(index int) (I, error) {
	var zero I
	if t == nil || t.root == nil {
		return zero, ErrIndexOutOfBounds
	}
	if index < 0 || index >= t.Len() {
		return zero, ErrIndexOutOfBounds
	}
	return t.atNode(t.root, t.height, index)
}

func (t *Tree[I, S]) atNode(n treeNode[I, S], height int, index int) (I, error) {
	var zero I
	assert(n != nil, "atNode called with nil node")
	assert(height > 0, "atNode called with non-positive height")
	if height == 1 {
		leaf := n.(*leafNode[I, S])
		if index < 0 || index >= len(leaf.items) {
			return zero, ErrIndexOutOfBounds
		}
		return leaf.items[index], nil
	}
	inner := n.(*innerNode[I, S])
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
