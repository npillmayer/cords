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
		return t, nil
	}
	cloned := t.Clone()
	for i, item := range items {
		if err := cloned.insertOneAt(index+i, item); err != nil {
			return nil, err
		}
	}
	return cloned, nil
}

// DeleteAt removes one item at index and returns a new tree.
//
// Delete uses recursive path-copy with sibling borrow/merge rebalancing.
func (t *Tree[I, S]) DeleteAt(index int) (*Tree[I, S], error) {
	if t == nil {
		return nil, fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	size := t.Len()
	if index < 0 || index >= size {
		return nil, ErrIndexOutOfBounds
	}
	cloned := t.Clone()
	needsRebalance, err := cloned.deleteOneAt(index)
	if err != nil {
		return nil, err
	}
	if needsRebalance {
		return nil, fmt.Errorf("%w: delete rebalance could not be resolved", ErrUnimplemented)
	}
	return cloned, nil
}

// DeleteRange removes count items starting at index and returns a new tree.
//
// This implementation is intentionally compositional: split at range start,
// delete from the right fragment, then concat.
func (t *Tree[I, S]) DeleteRange(index, count int) (*Tree[I, S], error) {
	if t == nil {
		return nil, fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	size := t.Len()
	if index < 0 || count < 0 || index > size || index+count > size {
		return nil, ErrIndexOutOfBounds
	}
	if count == 0 {
		return t, nil
	}
	left, right, err := t.SplitAt(index)
	if err != nil {
		return nil, err
	}
	trimmed := right
	for i := 0; i < count; i++ {
		trimmed, err = trimmed.DeleteAt(0)
		if err != nil {
			return nil, err
		}
	}
	out, err := left.Concat(trimmed)
	if err != nil {
		return nil, err
	}
	return out, nil
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
		return t, t, nil
	}
	if index == 0 {
		empty, err := New[I, S](t.cfg)
		if err != nil {
			return nil, nil, err
		}
		return empty, t, nil
	}
	if index == size {
		empty, err := New[I, S](t.cfg)
		if err != nil {
			return nil, nil, err
		}
		return t, empty, nil
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
	return left, right, nil
}

// Concat concatenates another tree and returns a new tree.
func (t *Tree[I, S]) Concat(other *Tree[I, S]) (*Tree[I, S], error) {
	if t == nil || other == nil {
		return nil, fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	if t.IsEmpty() {
		return other, nil
	}
	if other.IsEmpty() {
		return t, nil
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

func (t *Tree[I, S]) deleteOneAt(index int) (needsRebalance bool, err error) {
	assert(t.root != nil, "deleteOneAt called on empty tree")
	updated, needsRebalance, err := t.deleteRecursive(t.root, t.height, index, true)
	if err != nil {
		return false, err
	}
	t.root = normalizeNode[I, S](updated)
	t.normalizeRoot()
	t.assertDeleteRootNormalized()
	return needsRebalance, nil
}

func (t *Tree[I, S]) assertDeleteRootNormalized() {
	if t.root == nil {
		assert(t.height == 0, "delete root normalization: nil root must have height 0")
		return
	}
	if t.root.isLeaf() {
		leaf := t.root.(*leafNode[I, S])
		assert(len(leaf.items) > 0, "delete root normalization: root leaf must be non-empty")
		assert(t.height == 1, "delete root normalization: root leaf must have height 1")
		return
	}
	inner := t.root.(*innerNode[I, S])
	assert(len(inner.children) > 1, "delete root normalization: root inner must have at least 2 children")
	assert(t.height >= 2, "delete root normalization: root inner must have height >= 2")
}

func (t *Tree[I, S]) deleteRecursive(
	n treeNode[I, S], height, index int, isRoot bool,
) (updated treeNode[I, S], needsRebalance bool, err error) {
	assert(n != nil, "deleteRecursive called with nil node")
	assert(height > 0, "deleteRecursive called with invalid height")
	if height == 1 {
		leaf, ok := n.(*leafNode[I, S])
		assert(ok, "deleteRecursive expected leaf at height 1")
		if index < 0 || index >= len(leaf.items) {
			return nil, false, ErrIndexOutOfBounds
		}
		cloned := t.cloneLeaf(leaf)
		t.removeLeafItemsRange(cloned, index, index+1)
		t.recomputeLeafSummary(cloned)
		if len(cloned.items) == 0 {
			if isRoot {
				return nil, false, nil
			}
			return cloned, true, nil
		}
		return cloned, !isRoot && t.leafUnderflow(cloned, false), nil
	}

	inner, ok := n.(*innerNode[I, S])
	assert(ok, "deleteRecursive expected internal node")
	cloned := t.cloneInner(inner)
	slot, localIndex, err := t.locateChildForDelete(cloned, index)
	if err != nil {
		return nil, false, err
	}
	updatedChild, childNeedsRebalance, err := t.deleteRecursive(cloned.children[slot], height-1, localIndex, false)
	if err != nil {
		return nil, false, err
	}
	updatedChild = normalizeNode[I, S](updatedChild)
	if updatedChild == nil {
		t.removeChildAt(cloned, slot)
	} else {
		cloned.children[slot] = updatedChild
		t.recomputeInnerSummary(cloned)
	}
	if childNeedsRebalance && updatedChild != nil {
		if !(isRoot && len(cloned.children) == 1) {
			resolved := t.rebalanceChildAfterDelete(cloned, slot, height-1)
			childNeedsRebalance = !resolved
		} else {
			childNeedsRebalance = false
		}
	}
	if len(cloned.children) == 0 {
		if isRoot {
			return nil, false, nil
		}
		return nil, true, nil
	}
	selfUnderflow := !isRoot && t.innerUnderflow(cloned, false)
	return cloned, childNeedsRebalance || selfUnderflow, nil
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

func (t *Tree[I, S]) locateChildForDelete(inner *innerNode[I, S], index int) (childSlot int, localIndex int, err error) {
	assert(inner != nil, "locateChildForDelete called with nil inner node")
	assert(len(inner.children) > 0, "locateChildForDelete called with empty children")
	if index < 0 {
		return 0, 0, ErrIndexOutOfBounds
	}
	remaining := index
	for i, child := range inner.children {
		childItems := t.countItems(child)
		if remaining < childItems {
			return i, remaining, nil
		}
		remaining -= childItems
	}
	return 0, 0, ErrIndexOutOfBounds
}

func (t *Tree[I, S]) rebalanceChildAfterDelete(parent *innerNode[I, S], slot int, childHeight int) bool {
	assert(parent != nil, "rebalanceChildAfterDelete called with nil parent")
	assert(slot >= 0 && slot < len(parent.children), "rebalanceChildAfterDelete slot out of range")
	assert(childHeight > 0, "rebalanceChildAfterDelete childHeight must be positive")
	child := parent.children[slot]
	assert(child != nil, "rebalanceChildAfterDelete child is nil")
	if childHeight == 1 {
		return t.rebalanceLeafChild(parent, slot)
	}
	return t.rebalanceInnerChild(parent, slot)
}

// applyRebalancePolicy centralizes sibling operation order after delete:
// borrow-left, borrow-right, merge-left, merge-right.
func (t *Tree[I, S]) applyRebalancePolicy(
	parent *innerNode[I, S], slot int,
	borrowLeft func() bool,
	borrowRight func() bool,
	mergeLeft func() bool,
	mergeRight func() bool,
) bool {
	assert(parent != nil, "applyRebalancePolicy called with nil parent")
	assert(slot >= 0 && slot < len(parent.children), "applyRebalancePolicy slot out of range")
	hasLeft := slot > 0
	hasRight := slot+1 < len(parent.children)
	if hasLeft && borrowLeft != nil && borrowLeft() {
		return true
	}
	if hasRight && borrowRight != nil && borrowRight() {
		return true
	}
	if hasLeft && mergeLeft != nil && mergeLeft() {
		return true
	}
	if hasRight && mergeRight != nil && mergeRight() {
		return true
	}
	return false
}

func (t *Tree[I, S]) rebalanceLeafChild(parent *innerNode[I, S], slot int) bool {
	child, ok := parent.children[slot].(*leafNode[I, S])
	assert(ok, "rebalanceLeafChild expected leaf child")
	if !t.leafUnderflow(child, false) {
		return true
	}
	return t.applyRebalancePolicy(
		parent, slot,
		func() bool {
			left, lok := parent.children[slot-1].(*leafNode[I, S])
			assert(lok, "rebalanceLeafChild expected leaf left sibling")
			if len(left.items) <= fixedBase {
				return false
			}
			leftClone := t.cloneLeaf(left)
			parent.children[slot-1] = leftClone
			borrowed := leftClone.items[len(leftClone.items)-1]
			t.removeLeafItemsRange(leftClone, len(leftClone.items)-1, len(leftClone.items))
			t.insertLeafItemsAt(child, 0, borrowed)
			t.recomputeLeafSummary(leftClone)
			t.recomputeLeafSummary(child)
			t.recomputeInnerSummary(parent)
			return true
		},
		func() bool {
			right, rok := parent.children[slot+1].(*leafNode[I, S])
			assert(rok, "rebalanceLeafChild expected leaf right sibling")
			if len(right.items) <= fixedBase {
				return false
			}
			rightClone := t.cloneLeaf(right)
			parent.children[slot+1] = rightClone
			borrowed := rightClone.items[0]
			t.removeLeafItemsRange(rightClone, 0, 1)
			t.insertLeafItemsAt(child, len(child.items), borrowed)
			t.recomputeLeafSummary(rightClone)
			t.recomputeLeafSummary(child)
			t.recomputeInnerSummary(parent)
			return true
		},
		func() bool {
			left, lok := parent.children[slot-1].(*leafNode[I, S])
			assert(lok, "rebalanceLeafChild expected leaf left sibling for merge")
			merged := make([]I, 0, len(left.items)+len(child.items))
			merged = append(merged, left.items...)
			merged = append(merged, child.items...)
			parent.children[slot-1] = t.makeLeaf(merged)
			t.removeChildAt(parent, slot)
			return true
		},
		func() bool {
			right, rok := parent.children[slot+1].(*leafNode[I, S])
			assert(rok, "rebalanceLeafChild expected leaf right sibling for merge")
			merged := make([]I, 0, len(child.items)+len(right.items))
			merged = append(merged, child.items...)
			merged = append(merged, right.items...)
			parent.children[slot] = t.makeLeaf(merged)
			t.removeChildAt(parent, slot+1)
			return true
		},
	)
}

func (t *Tree[I, S]) rebalanceInnerChild(parent *innerNode[I, S], slot int) bool {
	child, ok := parent.children[slot].(*innerNode[I, S])
	assert(ok, "rebalanceInnerChild expected internal child")
	if !t.innerUnderflow(child, false) {
		return true
	}
	return t.applyRebalancePolicy(
		parent, slot,
		func() bool {
			left, lok := parent.children[slot-1].(*innerNode[I, S])
			assert(lok, "rebalanceInnerChild expected internal left sibling")
			if len(left.children) <= fixedBase {
				return false
			}
			leftClone := t.cloneInner(left)
			parent.children[slot-1] = leftClone
			borrowed := leftClone.children[len(leftClone.children)-1]
			t.removeChildAt(leftClone, len(leftClone.children)-1)
			t.insertChildAt(child, 0, borrowed)
			t.recomputeInnerSummary(parent)
			return true
		},
		func() bool {
			right, rok := parent.children[slot+1].(*innerNode[I, S])
			assert(rok, "rebalanceInnerChild expected internal right sibling")
			if len(right.children) <= fixedBase {
				return false
			}
			rightClone := t.cloneInner(right)
			parent.children[slot+1] = rightClone
			borrowed := rightClone.children[0]
			t.removeChildAt(rightClone, 0)
			t.insertChildAt(child, len(child.children), borrowed)
			t.recomputeInnerSummary(parent)
			return true
		},
		func() bool {
			left, lok := parent.children[slot-1].(*innerNode[I, S])
			assert(lok, "rebalanceInnerChild expected internal left sibling for merge")
			mergedChildren := make([]treeNode[I, S], 0, len(left.children)+len(child.children))
			mergedChildren = append(mergedChildren, left.children...)
			mergedChildren = append(mergedChildren, child.children...)
			parent.children[slot-1] = t.makeInternal(mergedChildren...)
			t.removeChildAt(parent, slot)
			return true
		},
		func() bool {
			right, rok := parent.children[slot+1].(*innerNode[I, S])
			assert(rok, "rebalanceInnerChild expected internal right sibling for merge")
			mergedChildren := make([]treeNode[I, S], 0, len(child.children)+len(right.children))
			mergedChildren = append(mergedChildren, child.children...)
			mergedChildren = append(mergedChildren, right.children...)
			parent.children[slot] = t.makeInternal(mergedChildren...)
			t.removeChildAt(parent, slot+1)
			return true
		},
	)
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
