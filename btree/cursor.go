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
		return nil, fmt.Errorf("%w: extension is nil", ErrInvalidConfig)
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
func (c *Cursor[I, S, E, K]) Seek(target K) (itemIndex int, acc K, err error) {
	if c == nil || c.tree == nil || c.dim == nil {
		var zero K
		return 0, zero, fmt.Errorf("%w: cursor not initialized", ErrInvalidDimension)
	}
	zero := c.dim.Zero()
	if c.dim.Compare(zero, target) >= 0 {
		return 0, zero, nil
	}
	if c.tree.root == nil {
		return 0, zero, nil
	}
	idx, reached, found, err := c.seekNode(c.tree.root, 0, zero, target)
	if err != nil {
		var z K
		return 0, z, err
	}
	if found {
		return idx, reached, nil
	}
	return c.tree.Len(), reached, nil
}

// Seek finds the first item index where accumulated extension dimension reaches target.
func (c *ExtCursor[I, S, E, K]) Seek(target K) (itemIndex int, acc K, err error) {
	if c == nil || c.tree == nil || c.dim == nil {
		var zero K
		return 0, zero, fmt.Errorf("%w: cursor not initialized", ErrInvalidDimension)
	}
	if c.tree.cfg.Extension == nil {
		var zero K
		return 0, zero, fmt.Errorf("%w: extension is nil", ErrInvalidConfig)
	}
	zero := c.dim.Zero()
	if c.dim.Compare(zero, target) >= 0 {
		return 0, zero, nil
	}
	if c.tree.root == nil {
		return 0, zero, nil
	}
	idx, reached, found, err := c.seekNode(c.tree.root, 0, zero, target)
	if err != nil {
		var z K
		return 0, z, err
	}
	if found {
		return idx, reached, nil
	}
	return c.tree.Len(), reached, nil
}

// seekNode descends to the first leaf position where accumulated dimension
// reaches target.
//
// `startIndex` and `acc` describe the prefix state before subtree n.
func (c *Cursor[I, S, E, K]) seekNode(n treeNode[I, S, E], startIndex int, acc K, target K) (idx int, reached K, found bool, err error) {
	assert(n != nil, "cursor seekNode called with nil node")
	if n.isLeaf() {
		leaf := n.(*leafNode[I, S, E])
		cur := acc
		for i, item := range leaf.items {
			next := c.dim.Add(cur, item.Summary())
			if c.dim.Compare(next, target) >= 0 {
				return startIndex + i, next, true, nil
			}
			cur = next
		}
		return startIndex + len(leaf.items), cur, false, nil
	}
	inner := n.(*innerNode[I, S, E])
	curIdx := startIndex
	curAcc := acc
	for _, child := range inner.children {
		assert(child != nil, "cursor seekNode encountered nil child")
		nextAcc := c.dim.Add(curAcc, child.Summary())
		if c.dim.Compare(nextAcc, target) >= 0 {
			return c.seekNode(child, curIdx, curAcc, target)
		}
		curAcc = nextAcc
		curIdx += c.tree.countItems(child)
	}
	return curIdx, curAcc, false, nil
}

func (c *ExtCursor[I, S, E, K]) seekNode(n treeNode[I, S, E], startIndex int, acc K, target K) (idx int, reached K, found bool, err error) {
	assert(n != nil, "ext cursor seekNode called with nil node")
	if n.isLeaf() {
		leaf := n.(*leafNode[I, S, E])
		cur := acc
		for i, item := range leaf.items {
			step := c.tree.cfg.Extension.FromItem(item, item.Summary())
			next := c.dim.Add(cur, step)
			if c.dim.Compare(next, target) >= 0 {
				return startIndex + i, next, true, nil
			}
			cur = next
		}
		return startIndex + len(leaf.items), cur, false, nil
	}
	inner := n.(*innerNode[I, S, E])
	curIdx := startIndex
	curAcc := acc
	for _, child := range inner.children {
		assert(child != nil, "ext cursor seekNode encountered nil child")
		nextAcc := c.dim.Add(curAcc, child.Ext())
		if c.dim.Compare(nextAcc, target) >= 0 {
			return c.seekNode(child, curIdx, curAcc, target)
		}
		curAcc = nextAcc
		curIdx += c.tree.countItems(child)
	}
	return curIdx, curAcc, false, nil
}
