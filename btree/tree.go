package btree

import (
	"fmt"
)

// Tree is a persistent, rope-oriented B+ sum-tree.
//
// I is the leaf item type (for ropes usually text chunks), S is the summary
// type aggregated through the tree. The item type is tied to summary type via
// SummarizedItem[S].
type Tree[I SummarizedItem[S], S any] struct {
	cfg    Config[S]
	root   treeNode[I, S]
	height int // 0 means empty tree
}

// New creates an empty tree with validated configuration.
func New[I SummarizedItem[S], S any](cfg Config[S]) (*Tree[I, S], error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg = cfg.normalized()
	return &Tree[I, S]{cfg: cfg}, nil
}

// Config returns a copy of the effective tree configuration.
func (t *Tree[I, S]) Config() Config[S] {
	return t.cfg
}

// Clone returns a shallow clone of the tree root container.
//
// Node contents are shared intentionally; mutating operations will use path-copy
// semantics once implemented.
func (t *Tree[I, S]) Clone() *Tree[I, S] {
	if t == nil {
		return nil
	}
	cloned := *t
	return &cloned
}

// IsEmpty reports whether the tree has no items.
func (t *Tree[I, S]) IsEmpty() bool {
	return t == nil || t.root == nil
}

// Len returns the number of items in the tree.
func (t *Tree[I, S]) Len() int {
	if t == nil || t.root == nil {
		return 0
	}
	return t.countItems(t.root)
}

// Height returns the tree height, where 0 means empty and 1 means a leaf root.
func (t *Tree[I, S]) Height() int {
	if t == nil {
		return 0
	}
	return t.height
}

// Summary returns the root summary, or Zero() for an empty tree.
func (t *Tree[I, S]) Summary() S {
	if t == nil || t.root == nil {
		return t.cfg.Monoid.Zero()
	}
	return t.root.Summary()
}

// InsertAt inserts items at an item index and returns a new tree.
func (t *Tree[I, S]) InsertAt(index int, items ...I) (*Tree[I, S], error) {
	if t == nil {
		return nil, fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	size := t.Len()
	if index < 0 || index > size {
		return nil, ErrIndexOutOfBounds
	}
	if len(items) == 0 {
		return t.Clone(), nil
	}
	cloned := t.Clone()
	for i, item := range items {
		if err := cloned.insertOneAt(index+i, item); err != nil {
			return nil, err
		}
	}
	return cloned, nil
}

// SplitAt splits a tree at an item index and returns left and right trees.
func (t *Tree[I, S]) SplitAt(index int) (*Tree[I, S], *Tree[I, S], error) {
	if t == nil {
		return nil, nil, fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	size := t.Len()
	if index < 0 || index > size {
		return nil, nil, ErrIndexOutOfBounds
	}
	if t.IsEmpty() {
		return t.Clone(), t.Clone(), nil
	}
	if index == 0 {
		empty, err := New[I, S](t.cfg)
		if err != nil {
			return nil, nil, err
		}
		return empty, t.Clone(), nil
	}
	if index == size {
		empty, err := New[I, S](t.cfg)
		if err != nil {
			return nil, nil, err
		}
		return t.Clone(), empty, nil
	}
	leftRoot, rightRoot, err := t.splitNodePathCopy(t.root, t.height, index)
	if err != nil {
		return nil, nil, err
	}
	left := t.Clone()
	right := t.Clone()
	left.root = leftRoot
	right.root = rightRoot
	left.height = left.subtreeHeight(left.root)
	right.height = right.subtreeHeight(right.root)
	left.normalizeRoot()
	right.normalizeRoot()
	assert(left.Check() == nil, "left tree is invalid")
	assert(right.Check() == nil, "right tree is invalid")
	return left, right, nil
}

// Concat concatenates another tree and returns a new tree.
func (t *Tree[I, S]) Concat(other *Tree[I, S]) (*Tree[I, S], error) {
	if t == nil || other == nil {
		return nil, fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	if t.IsEmpty() {
		return other.Clone(), nil
	}
	if other.IsEmpty() {
		return t.Clone(), nil
	}
	left, right, height, err := t.concatNodes(t.root, t.height, other.root, other.height)
	if err != nil {
		return nil, err
	}
	combined := t.Clone()
	left = normalizeNode[I, S](left)
	right = normalizeNode[I, S](right)
	if left == nil {
		combined.root = right
		combined.height = height
		combined.normalizeRoot()
		return combined, nil
	}
	if right == nil {
		combined.root = left
		combined.height = height
		combined.normalizeRoot()
		return combined, nil
	}
	combined.root = t.makeInternal(left, right)
	combined.height = height + 1
	combined.normalizeRoot()
	return combined, nil
}

func (t *Tree[I, S]) concatNodes(
	left treeNode[I, S], leftHeight int,
	right treeNode[I, S], rightHeight int,
) (mergedLeft treeNode[I, S], mergedRight treeNode[I, S], outHeight int, err error) {
	left = normalizeNode[I, S](left)
	right = normalizeNode[I, S](right)
	switch {
	case left == nil && right == nil:
		return nil, nil, 0, nil
	case left == nil:
		return right, nil, rightHeight, nil
	case right == nil:
		return left, nil, leftHeight, nil
	}

	if leftHeight == rightHeight {
		l, r, joinErr := t.concatSameHeight(left, right, leftHeight)
		if joinErr != nil {
			return nil, nil, 0, joinErr
		}
		return normalizeNode[I, S](l), normalizeNode[I, S](r), leftHeight, nil
	}

	if leftHeight > rightHeight {
		inner, ok := left.(*innerNode[I, S])
		assert(ok, "concatNodes expected internal left node at greater height")
		cloned := t.cloneInner(inner)
		last := len(cloned.children) - 1
		childLeft, childRight, _, joinErr := t.concatNodes(cloned.children[last], leftHeight-1, right, rightHeight)
		if joinErr != nil {
			return nil, nil, 0, joinErr
		}
		cloned.children[last] = childLeft
		childRight = normalizeNode[I, S](childRight)
		if childRight != nil {
			t.insertChildAt(cloned, last+1, childRight)
		} else {
			t.recomputeInnerSummary(cloned)
		}
		if t.innerOverflow(cloned) {
			l, r, splitErr := t.splitInner(cloned)
			if splitErr != nil {
				assert(false, splitErr.Error())
			}
			return l, r, leftHeight, nil
		}
		return cloned, nil, leftHeight, nil
	}

	inner, ok := right.(*innerNode[I, S])
	assert(ok, "concatNodes expected internal right node at greater height")
	cloned := t.cloneInner(inner)
	first := 0
	childLeft, childRight, _, joinErr := t.concatNodes(left, leftHeight, cloned.children[first], rightHeight-1)
	if joinErr != nil {
		return nil, nil, 0, joinErr
	}
	cloned.children[first] = childLeft
	childRight = normalizeNode[I, S](childRight)
	if childRight != nil {
		t.insertChildAt(cloned, 1, childRight)
	} else {
		t.recomputeInnerSummary(cloned)
	}
	if t.innerOverflow(cloned) {
		l, r, splitErr := t.splitInner(cloned)
		if splitErr != nil {
			assert(false, splitErr.Error())
		}
		return l, r, rightHeight, nil
	}
	return cloned, nil, rightHeight, nil
}

func (t *Tree[I, S]) concatSameHeight(left, right treeNode[I, S], height int) (treeNode[I, S], treeNode[I, S], error) {
	assert(height > 0, "concatSameHeight called with non-positive height")
	if height == 1 {
		leftLeaf, lok := left.(*leafNode[I, S])
		rightLeaf, rok := right.(*leafNode[I, S])
		assert(lok && rok, "concatSameHeight expected leaf nodes at height 1")
		total := len(leftLeaf.items) + len(rightLeaf.items)
		if total <= fixedMaxLeafItems {
			merged := make([]I, 0, total)
			merged = append(merged, leftLeaf.items...)
			merged = append(merged, rightLeaf.items...)
			return t.makeLeaf(merged), nil, nil
		}
		return left, right, nil
	}
	leftInner, lok := left.(*innerNode[I, S])
	rightInner, rok := right.(*innerNode[I, S])
	assert(lok && rok, "concatSameHeight expected internal nodes")
	total := len(leftInner.children) + len(rightInner.children)
	if total <= fixedMaxChildren {
		children := make([]treeNode[I, S], 0, total)
		children = append(children, leftInner.children...)
		children = append(children, rightInner.children...)
		return t.makeInternal(children...), nil, nil
	}
	return left, right, nil
}

func (t *Tree[I, S]) countItems(n treeNode[I, S]) int {
	if n == nil {
		return 0
	}
	if n.isLeaf() {
		return len(n.(*leafNode[I, S]).items)
	}
	total := 0
	for _, child := range n.(*innerNode[I, S]).children {
		total += t.countItems(child)
	}
	return total
}

func (t *Tree[I, S]) splitNodePathCopy(n treeNode[I, S], height, index int) (treeNode[I, S], treeNode[I, S], error) {
	if n == nil {
		assert(index == 0, "splitNodePathCopy called with nil node and non-zero index")
		return nil, nil, nil
	}
	total := t.countItems(n)
	assert(index >= 0 && index <= total, "splitNodePathCopy index out of bounds")
	if index == 0 {
		return nil, n, nil
	}
	if index == total {
		return n, nil, nil
	}
	if height == 1 {
		leaf, ok := n.(*leafNode[I, S])
		assert(ok, "splitNodePathCopy expected leaf at height 1")
		left := t.makeLeaf(leaf.items[:index])
		right := t.makeLeaf(leaf.items[index:])
		return left, right, nil
	}
	inner, ok := n.(*innerNode[I, S])
	assert(ok, "splitNodePathCopy expected internal node")
	slot, local, err := t.locateChildForInsert(inner, index)
	if err != nil {
		assert(false, err.Error())
	}
	childLeft, childRight, err := t.splitNodePathCopy(inner.children[slot], height-1, local)
	if err != nil {
		assert(false, err.Error())
	}
	var leftNode treeNode[I, S]
	var rightNode treeNode[I, S]
	leftChildren := make([]treeNode[I, S], 0, slot+1)
	leftChildren = append(leftChildren, inner.children[:slot]...)
	childLeft = normalizeNode[I, S](childLeft)
	if childLeft != nil {
		leftChildren = append(leftChildren, childLeft)
	}
	if len(leftChildren) > 0 {
		leftNode = t.makeInternal(leftChildren...)
	}
	rightChildren := make([]treeNode[I, S], 0, len(inner.children)-slot)
	childRight = normalizeNode[I, S](childRight)
	if childRight != nil {
		rightChildren = append(rightChildren, childRight)
	}
	rightChildren = append(rightChildren, inner.children[slot+1:]...)
	if len(rightChildren) > 0 {
		rightNode = t.makeInternal(rightChildren...)
	}
	return leftNode, rightNode, nil
}

func (t *Tree[I, S]) subtreeHeight(n treeNode[I, S]) int {
	h := 0
	cur := normalizeNode[I, S](n)
	for cur != nil {
		h++
		if cur.isLeaf() {
			return h
		}
		inner := cur.(*innerNode[I, S])
		if len(inner.children) == 0 {
			return h
		}
		cur = normalizeNode[I, S](inner.children[0])
	}
	return 0
}

func (t *Tree[I, S]) normalizeRoot() {
	if t == nil {
		return
	}
	t.root = normalizeNode[I, S](t.root)
	if t.root == nil {
		t.height = 0
		return
	}
	for {
		inner, ok := t.root.(*innerNode[I, S])
		if !ok {
			t.height = 1
			return
		}
		if len(inner.children) != 1 {
			if t.height == 0 {
				t.height = t.subtreeHeight(t.root)
			}
			return
		}
		t.root = normalizeNode[I, S](inner.children[0])
		if t.height > 0 {
			t.height--
		}
		if t.root == nil {
			t.height = 0
			return
		}
	}
}

func (t *Tree[I, S]) insertOneAt(index int, item I) error {
	if t.root == nil {
		t.root = t.makeLeaf([]I{item})
		t.height = 1
		return nil
	}
	updated, promoted, err := t.insertRecursive(t.root, t.height, index, item)
	if err != nil {
		return err
	}
	promoted = normalizeNode[I, S](promoted)
	if promoted != nil {
		t.root = t.makeInternal(updated, promoted)
		t.height++
		return nil
	}
	t.root = updated
	return nil
}

func (t *Tree[I, S]) insertRecursive(n treeNode[I, S], height, index int, item I) (treeNode[I, S], treeNode[I, S], error) {
	assert(n != nil, "insertRecursive called with nil node")
	assert(height > 0, "insertRecursive called with invalid height")
	if height == 1 {
		leaf, ok := n.(*leafNode[I, S])
		assert(ok, "insertRecursive expected leaf at height 1")
		left, right, err := t.insertIntoLeafLocal(leaf, index, item)
		if err != nil {
			assert(false, err.Error())
		}
		return left, normalizeNode[I, S](right), nil
	}

	inner, ok := n.(*innerNode[I, S])
	assert(ok, "insertRecursive expected internal node")
	cloned := t.cloneInner(inner)
	slot, localIndex, err := t.locateChildForInsert(cloned, index)
	if err != nil {
		assert(false, err.Error())
	}
	updatedChild, promotedChild, err := t.insertRecursive(cloned.children[slot], height-1, localIndex, item)
	if err != nil {
		assert(false, err.Error())
	}
	promotedChild = normalizeNode[I, S](promotedChild)
	cloned.children[slot] = updatedChild
	if promotedChild != nil {
		t.insertChildAt(cloned, slot+1, promotedChild)
	} else {
		t.recomputeInnerSummary(cloned)
	}
	if !t.innerOverflow(cloned) {
		return cloned, nil, nil
	}
	left, right, err := t.splitInner(cloned)
	if err != nil {
		assert(false, err.Error())
	}
	return left, normalizeNode[I, S](right), nil
}

func (t *Tree[I, S]) locateChildForInsert(inner *innerNode[I, S], index int) (childSlot int, localIndex int, err error) {
	assert(inner != nil, "locateChildForInsert called with nil inner node")
	assert(len(inner.children) > 0, "locateChildForInsert called with empty children")
	assert(index >= 0, "locateChildForInsert called with negative index")
	remaining := index
	for i, child := range inner.children {
		childItems := t.countItems(child)
		if remaining <= childItems {
			return i, remaining, nil
		}
		remaining -= childItems
	}
	assert(false, "locateChildForInsert index exceeded subtree item count")
	return 0, 0, nil
}

func (t *Tree[I, S]) splitInner(inner *innerNode[I, S]) (*innerNode[I, S], *innerNode[I, S], error) {
	assert(inner != nil, "splitInner called with nil inner node")
	n := len(inner.children)
	maxChildren := fixedMaxChildren
	if n <= maxChildren {
		return t.cloneInner(inner), nil, nil
	}
	assert(n <= 2*maxChildren, "splitInner requires more than one promoted sibling")
	mid := n / 2
	leftChildren := append([]treeNode[I, S](nil), inner.children[:mid]...)
	rightChildren := append([]treeNode[I, S](nil), inner.children[mid:]...)
	left := t.makeInternal(leftChildren...)
	right := t.makeInternal(rightChildren...)
	assert(len(left.children) >= fixedBase && len(right.children) >= fixedBase,
		"splitInner violates internal occupancy bounds")
	return left, right, nil
}

func normalizeNode[I SummarizedItem[S], S any](n treeNode[I, S]) treeNode[I, S] {
	switch v := n.(type) {
	case nil:
		return nil
	case *leafNode[I, S]:
		if v == nil {
			return nil
		}
	case *innerNode[I, S]:
		if v == nil {
			return nil
		}
	}
	return n
}
