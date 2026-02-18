package btree

import "fmt"

// Check validates structural tree invariants.
//
// This checker is intentionally strict and should be used in tests while the
// implementation is evolving.
func (t *Tree[I, S]) Check() error {
	if t == nil {
		return fmt.Errorf("%w: nil tree", ErrInvalidConfig)
	}
	if t.root == nil {
		if t.height != 0 {
			return fmt.Errorf("%w: empty tree must have height=0", ErrInvalidConfig)
		}
		return nil
	}
	if t.height <= 0 {
		return fmt.Errorf("%w: non-empty tree must have height > 0", ErrInvalidConfig)
	}
	_, height, err := t.checkNode(t.root, true)
	if err != nil {
		return err
	}
	if height != t.height {
		return fmt.Errorf("%w: height mismatch (%d != %d)", ErrInvalidConfig, height, t.height)
	}
	return nil
}

func (t *Tree[I, S]) checkNode(n treeNode[I, S], isRoot bool) (items int, height int, err error) {
	if n == nil {
		return 0, 0, fmt.Errorf("%w: nil node", ErrInvalidConfig)
	}
	if n.isLeaf() {
		leaf := n.(*leafNode[I, S])
		if leaf == nil {
			return 0, 0, fmt.Errorf("%w: nil leaf node", ErrInvalidConfig)
		}
		if err := t.checkBackendLeafInvariants(leaf); err != nil {
			return 0, 0, err
		}
		return len(leaf.items), 1, nil
	}
	inner := n.(*innerNode[I, S])
	if err := t.checkBackendInnerInvariants(inner); err != nil {
		return 0, 0, err
	}
	if len(inner.children) == 0 {
		return 0, 0, fmt.Errorf("%w: internal node has no children", ErrInvalidConfig)
	}
	if !isRoot {
		if len(inner.children) > fixedMaxChildren {
			return 0, 0, fmt.Errorf("%w: child count %d exceeds degree %d",
				ErrInvalidConfig, len(inner.children), fixedMaxChildren)
		}
	}
	var totalItems int
	var childHeight int
	for i, child := range inner.children {
		if child == nil {
			return 0, 0, fmt.Errorf("%w: nil child at index %d", ErrInvalidConfig, i)
		}
		cItems, cHeight, cErr := t.checkNode(child, false)
		if cErr != nil {
			return 0, 0, cErr
		}
		totalItems += cItems
		if i == 0 {
			childHeight = cHeight
		} else if cHeight != childHeight {
			return 0, 0, fmt.Errorf("%w: non-uniform subtree heights", ErrInvalidConfig)
		}
	}
	return totalItems, childHeight + 1, nil
}
