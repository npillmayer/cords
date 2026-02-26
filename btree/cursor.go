package btree

import "fmt"

// Dimension describes a seek dimension over summaries.
//
// K is the dimension key/position type.
type Dimension[S any, K any] interface {
	Zero() K
	Add(acc K, summary S) K
	Compare(acc K, target K) int
}

// Cursor tracks a seek position in a tree along a given dimension.
type Cursor[I SummarizedItem[S], S, E any, K any] struct {
	tree *Tree[I, S, E]
	dim  Dimension[S, K]
}

// ExtCursor tracks a seek position in a tree along an extension-based dimension.
type ExtCursor[I SummarizedItem[S], S, E any, K any] struct {
	tree *Tree[I, S, E]
	dim  Dimension[E, K]
}

type seekOps[I SummarizedItem[S], S, E any, K any] struct {
	zero     K
	compare  func(K, K) int
	addItem  func(K, I) K
	addChild func(K, treeNode[I, S, E]) K
}

// NewCursor creates a cursor for a tree and a dimension.
func NewCursor[I SummarizedItem[S], S, E any, K any](tree *Tree[I, S, E], dim Dimension[S, K]) (*Cursor[I, S, E, K], error) {
	if tree == nil {
		return nil, fmt.Errorf("%w: tree is nil", ErrInvalidConfig)
	}
	if dim == nil {
		return nil, fmt.Errorf("%w: dimension is nil", ErrInvalidDimension)
	}
	return &Cursor[I, S, E, K]{
		tree: tree,
		dim:  dim,
	}, nil
}

// NewExtCursor creates a cursor over extension values for a tree and a dimension.
func NewExtCursor[I SummarizedItem[S], S, E any, K any](tree *Tree[I, S, E], dim Dimension[E, K]) (*ExtCursor[I, S, E, K], error) {
	if tree == nil {
		return nil, fmt.Errorf("%w: tree is nil", ErrInvalidConfig)
	}
	if tree.cfg.Extension == nil {
		return nil, fmt.Errorf("%w: extension is nil", ErrExtensionUnavailable)
	}
	if dim == nil {
		return nil, fmt.Errorf("%w: dimension is nil", ErrInvalidDimension)
	}
	return &ExtCursor[I, S, E, K]{
		tree: tree,
		dim:  dim,
	}, nil
}

// Seek finds the first item index where accumulated dimension reaches target.
//
// For target <= Zero() or an empty tree, Seek returns (0, Zero(), nil). If the
// target is beyond the total accumulated dimension, Seek returns (Len(), total,
// nil).
func (c *Cursor[I, S, E, K]) Seek(target K) (itemIndex int64, acc K, err error) {
	if c == nil || c.tree == nil || c.dim == nil {
		var zero K
		return 0, zero, fmt.Errorf("%w: cursor not initialized", ErrInvalidDimension)
	}
	ops := seekOps[I, S, E, K]{
		zero:    c.dim.Zero(),
		compare: c.dim.Compare,
		addItem: func(acc K, item I) K {
			return c.dim.Add(acc, item.Summary())
		},
		addChild: func(acc K, child treeNode[I, S, E]) K {
			return c.dim.Add(acc, child.Summary())
		},
	}
	return seekWithOps(c.tree, target, ops)
}

// SeekItem finds the first item where accumulated dimension reaches target.
//
// Returns found=false when target is at/before Zero(), when the tree is empty,
// or when target is beyond the total accumulated dimension.
func (c *Cursor[I, S, E, K]) SeekItem(target K) (itemIndex int64, item I, acc K, found bool, err error) {
	if c == nil || c.tree == nil || c.dim == nil {
		var zeroI I
		var zeroK K
		return 0, zeroI, zeroK, false, fmt.Errorf("%w: cursor not initialized", ErrInvalidDimension)
	}
	ops := seekOps[I, S, E, K]{
		zero:    c.dim.Zero(),
		compare: c.dim.Compare,
		addItem: func(acc K, item I) K {
			return c.dim.Add(acc, item.Summary())
		},
		addChild: func(acc K, child treeNode[I, S, E]) K {
			return c.dim.Add(acc, child.Summary())
		},
	}
	return seekItemWithOps(c.tree, target, ops)
}

// Seek finds the first item index where accumulated extension dimension reaches target.
//
// For target <= Zero() or an empty tree, Seek returns (0, Zero(), nil). If the
// target is beyond the total accumulated extension dimension, Seek returns
// (Len(), total, nil).
func (c *ExtCursor[I, S, E, K]) Seek(target K) (itemIndex int64, acc K, err error) {
	if c == nil || c.tree == nil || c.dim == nil {
		var zero K
		return 0, zero, fmt.Errorf("%w: cursor not initialized", ErrInvalidDimension)
	}
	if c.tree.cfg.Extension == nil {
		var zero K
		return 0, zero, fmt.Errorf("%w: extension is nil", ErrExtensionUnavailable)
	}
	ops := seekOps[I, S, E, K]{
		zero:    c.dim.Zero(),
		compare: c.dim.Compare,
		addItem: func(acc K, item I) K {
			step := c.tree.cfg.Extension.FromItem(item, item.Summary())
			return c.dim.Add(acc, step)
		},
		addChild: func(acc K, child treeNode[I, S, E]) K {
			return c.dim.Add(acc, child.Ext())
		},
	}
	return seekWithOps(c.tree, target, ops)
}

// SeekItem finds the first item where accumulated extension dimension reaches target.
//
// Returns found=false when target is at/before Zero(), when the tree is empty,
// or when target is beyond the total accumulated extension dimension.
func (c *ExtCursor[I, S, E, K]) SeekItem(target K) (itemIndex int64, item I, acc K, found bool, err error) {
	if c == nil || c.tree == nil || c.dim == nil {
		var zeroI I
		var zeroK K
		return 0, zeroI, zeroK, false, fmt.Errorf("%w: cursor not initialized", ErrInvalidDimension)
	}
	if c.tree.cfg.Extension == nil {
		var zeroI I
		var zeroK K
		return 0, zeroI, zeroK, false, fmt.Errorf("%w: extension is nil", ErrExtensionUnavailable)
	}
	ops := seekOps[I, S, E, K]{
		zero:    c.dim.Zero(),
		compare: c.dim.Compare,
		addItem: func(acc K, item I) K {
			step := c.tree.cfg.Extension.FromItem(item, item.Summary())
			return c.dim.Add(acc, step)
		},
		addChild: func(acc K, child treeNode[I, S, E]) K {
			return c.dim.Add(acc, child.Ext())
		},
	}
	return seekItemWithOps(c.tree, target, ops)
}

func seekWithOps[I SummarizedItem[S], S, E any, K any](tree *Tree[I, S, E], target K,
	ops seekOps[I, S, E, K]) (itemIndex int64, acc K, err error) {
	//
	if tree.root == nil {
		return 0, ops.zero, nil
	}
	if ops.compare(ops.zero, target) >= 0 {
		return 0, ops.zero, nil
	}
	idx, reached, found, err := seekNodeWithOps(tree, tree.root, 0, ops.zero, target, ops)
	if err != nil {
		var z K
		return 0, z, err
	}
	if found {
		return idx, reached, nil
	}
	return tree.Len(), reached, nil
}

func seekItemWithOps[I SummarizedItem[S], S, E any, K any](
	tree *Tree[I, S, E], target K, ops seekOps[I, S, E, K]) (
	itemIndex int64, item I, acc K, found bool, err error) {
	//
	var zeroI I
	if tree.root == nil {
		return 0, zeroI, ops.zero, false, nil
	}
	if ops.compare(ops.zero, target) >= 0 {
		return 0, zeroI, ops.zero, false, nil
	}
	idx, foundItem, reached, found, err := seekNodeItemWithOps(tree, tree.root, 0, ops.zero, target, ops)
	if err != nil {
		var z K
		return 0, zeroI, z, false, err
	}
	if found {
		return idx, foundItem, reached, true, nil
	}
	return tree.Len(), zeroI, reached, false, nil
}

// seekNodeWithOps descends to the first leaf position where accumulated dimension
// reaches target.
//
// `startIndex` and `acc` describe the prefix state before subtree n.
func seekNodeWithOps[I SummarizedItem[S], S, E any, K any](tree *Tree[I, S, E],
	n treeNode[I, S, E], startIndex int64, acc K, target K, ops seekOps[I, S, E, K]) (
	idx int64, reached K, found bool, err error) {
	//
	assert(n != nil, "seekNodeWithOps called with nil node")
	if n.isLeaf() {
		leaf := n.(*leafNode[I, S, E])
		cur := acc
		for i, item := range leaf.items {
			next := ops.addItem(cur, item)
			if ops.compare(next, target) >= 0 {
				return startIndex + int64(i), next, true, nil
			}
			cur = next
		}
		return startIndex + int64(len(leaf.items)), cur, false, nil
	}
	inner := n.(*innerNode[I, S, E])
	curIdx := startIndex
	curAcc := acc
	for _, child := range inner.children {
		assert(child != nil, "seekNodeWithOps encountered nil child")
		nextAcc := ops.addChild(curAcc, child)
		if ops.compare(nextAcc, target) >= 0 {
			return seekNodeWithOps(tree, child, curIdx, curAcc, target, ops)
		}
		curAcc = nextAcc
		curIdx += tree.countItems(child)
	}
	return curIdx, curAcc, false, nil
}

// seekNodeItemWithOps descends to the first leaf item where accumulated dimension
// reaches target.
//
// `startIndex` and `acc` describe the prefix state before subtree n.
func seekNodeItemWithOps[I SummarizedItem[S], S, E any, K any](
	tree *Tree[I, S, E], n treeNode[I, S, E], startIndex int64, acc K, target K, ops seekOps[I, S, E, K]) (
	idx int64, item I, reached K, found bool, err error) {
	//
	assert(n != nil, "seekNodeItemWithOps called with nil node")
	var zeroI I
	if n.isLeaf() {
		leaf := n.(*leafNode[I, S, E])
		cur := acc
		for i, entry := range leaf.items {
			next := ops.addItem(cur, entry)
			if ops.compare(next, target) >= 0 {
				return startIndex + int64(i), entry, next, true, nil
			}
			cur = next
		}
		// remove this
		//return startIndex + len(leaf.items), zeroI, cur, false, nil
		return startIndex + leaf.Weight(), zeroI, cur, false, nil
	}
	inner := n.(*innerNode[I, S, E])
	curIdx := startIndex
	curAcc := acc
	for _, child := range inner.children {
		assert(child != nil, "seekNodeItemWithOps encountered nil child")
		nextAcc := ops.addChild(curAcc, child)
		if ops.compare(nextAcc, target) >= 0 {
			return seekNodeItemWithOps(tree, child, curIdx, curAcc, target, ops)
		}
		curAcc = nextAcc
		curIdx += tree.countItems(child)
	}
	return curIdx, zeroI, curAcc, false, nil
}
