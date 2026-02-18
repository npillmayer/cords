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
	if errL, errR := left.Check(), right.Check(); errL == nil && errR == nil {
		return left, right, nil
	}
	// Fallback for edge cases where strict min-occupancy invariants reject a
	// structurally shared split result. This preserves correctness.
	items := t.itemsInOrder()
	left, err = t.buildFromItems(items[:index])
	if err != nil {
		return nil, nil, err
	}
	right, err = t.buildFromItems(items[index:])
	if err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

// Concat concatenates another tree and returns a new tree.
func (t *Tree[I, S]) Concat(other *Tree[I, S]) (*Tree[I, S], error) {
	if t == nil || other == nil {
		return nil, fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	if t.cfg.Degree != other.cfg.Degree || t.cfg.MinFill != other.cfg.MinFill {
		return nil, fmt.Errorf("%w: incompatible configs", ErrInvalidConfig)
	}
	if t.IsEmpty() {
		return other.Clone(), nil
	}
	if other.IsEmpty() {
		return t.Clone(), nil
	}
	combined := t.Clone()
	appendAt := combined.Len()
	for _, item := range other.itemsInOrder() {
		if err := combined.insertOneAt(appendAt, item); err != nil {
			return nil, err
		}
		appendAt++
	}
	return combined, nil
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

func (t *Tree[I, S]) itemsInOrder() []I {
	if t == nil || t.root == nil {
		return nil
	}
	out := make([]I, 0, t.Len())
	t.appendItems(t.root, &out)
	return out
}

func (t *Tree[I, S]) appendItems(n treeNode[I, S], out *[]I) {
	if n == nil {
		return
	}
	if n.isLeaf() {
		*out = append(*out, n.(*leafNode[I, S]).items...)
		return
	}
	for _, child := range n.(*innerNode[I, S]).children {
		t.appendItems(child, out)
	}
}

func (t *Tree[I, S]) buildFromItems(items []I) (*Tree[I, S], error) {
	tree, err := New[I, S](t.cfg)
	if err != nil {
		return nil, err
	}
	for i, item := range items {
		if err := tree.insertOneAt(i, item); err != nil {
			return nil, err
		}
	}
	return tree, nil
}

func (t *Tree[I, S]) splitNodePathCopy(n treeNode[I, S], height, index int) (treeNode[I, S], treeNode[I, S], error) {
	if n == nil {
		if index == 0 {
			return nil, nil, nil
		}
		return nil, nil, ErrIndexOutOfBounds
	}
	total := t.countItems(n)
	if index < 0 || index > total {
		return nil, nil, ErrIndexOutOfBounds
	}
	if index == 0 {
		return nil, n, nil
	}
	if index == total {
		return n, nil, nil
	}
	if height == 1 {
		leaf, ok := n.(*leafNode[I, S])
		if !ok {
			return nil, nil, fmt.Errorf("%w: expected leaf at height 1", ErrInvalidConfig)
		}
		left := t.makeLeaf(leaf.items[:index])
		right := t.makeLeaf(leaf.items[index:])
		return left, right, nil
	}
	inner, ok := n.(*innerNode[I, S])
	if !ok {
		return nil, nil, fmt.Errorf("%w: expected internal node at height %d", ErrInvalidConfig, height)
	}
	slot, local, err := t.locateChildForInsert(inner, index)
	if err != nil {
		return nil, nil, err
	}
	childLeft, childRight, err := t.splitNodePathCopy(inner.children[slot], height-1, local)
	if err != nil {
		return nil, nil, err
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
	if n == nil {
		return nil, nil, fmt.Errorf("%w: nil node", ErrInvalidConfig)
	}
	if height <= 0 {
		return nil, nil, fmt.Errorf("%w: invalid height", ErrInvalidConfig)
	}
	if height == 1 {
		leaf, ok := n.(*leafNode[I, S])
		if !ok {
			return nil, nil, fmt.Errorf("%w: expected leaf at height 1", ErrInvalidConfig)
		}
		left, right, err := t.insertIntoLeafLocal(leaf, index, item)
		if err != nil {
			return nil, nil, err
		}
		return left, normalizeNode[I, S](right), nil
	}

	inner, ok := n.(*innerNode[I, S])
	if !ok {
		return nil, nil, fmt.Errorf("%w: expected internal node at height %d", ErrInvalidConfig, height)
	}
	cloned := t.cloneInner(inner)
	slot, localIndex, err := t.locateChildForInsert(cloned, index)
	if err != nil {
		return nil, nil, err
	}
	updatedChild, promotedChild, err := t.insertRecursive(cloned.children[slot], height-1, localIndex, item)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	return left, normalizeNode[I, S](right), nil
}

func (t *Tree[I, S]) locateChildForInsert(inner *innerNode[I, S], index int) (childSlot int, localIndex int, err error) {
	if inner == nil || len(inner.children) == 0 {
		return 0, 0, fmt.Errorf("%w: internal node has no children", ErrInvalidConfig)
	}
	if index < 0 {
		return 0, 0, ErrIndexOutOfBounds
	}
	remaining := index
	for i, child := range inner.children {
		childItems := t.countItems(child)
		if remaining <= childItems {
			return i, remaining, nil
		}
		remaining -= childItems
	}
	return 0, 0, ErrIndexOutOfBounds
}

func (t *Tree[I, S]) splitInner(inner *innerNode[I, S]) (*innerNode[I, S], *innerNode[I, S], error) {
	if inner == nil {
		return nil, nil, fmt.Errorf("%w: nil inner node", ErrInvalidConfig)
	}
	n := len(inner.children)
	maxChildren := t.maxChildren()
	if n <= maxChildren {
		return t.cloneInner(inner), nil, nil
	}
	if n > 2*maxChildren {
		return nil, nil, fmt.Errorf("%w: internal split requires more than one sibling", ErrUnimplemented)
	}
	mid := n / 2
	leftChildren := append([]treeNode[I, S](nil), inner.children[:mid]...)
	rightChildren := append([]treeNode[I, S](nil), inner.children[mid:]...)
	left := t.makeInternal(leftChildren...)
	right := t.makeInternal(rightChildren...)
	if len(left.children) < t.minChildren() || len(right.children) < t.minChildren() {
		return nil, nil, fmt.Errorf("%w: split violates internal occupancy bounds", ErrInvalidConfig)
	}
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
