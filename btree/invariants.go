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
		if err := t.checkLeafInvariants(leaf); err != nil {
			return 0, 0, err
		}
		if isRoot {
			if len(leaf.items) == 0 {
				return 0, 0, fmt.Errorf("%w: root leaf must not be empty", ErrInvalidConfig)
			}
		} else {
			if len(leaf.items) < Base {
				return 0, 0, fmt.Errorf("%w: leaf underflow (%d < %d)",
					ErrInvalidConfig, len(leaf.items), Base)
			}
			if len(leaf.items) > MaxLeafItems {
				return 0, 0, fmt.Errorf("%w: leaf overflow (%d > %d)",
					ErrInvalidConfig, len(leaf.items), MaxLeafItems)
			}
		}
		return len(leaf.items), 1, nil
	}
	inner := n.(*innerNode[I, S])
	if err := t.checkInnerInvariants(inner); err != nil {
		return 0, 0, err
	}
	if len(inner.children) == 0 {
		return 0, 0, fmt.Errorf("%w: internal node has no children", ErrInvalidConfig)
	}
	if isRoot {
		if len(inner.children) == 1 {
			return 0, 0, fmt.Errorf("%w: root has a single child and should be collapsed", ErrInvalidConfig)
		}
	} else {
		if len(inner.children) < Base {
			return 0, 0, fmt.Errorf("%w: child count %d under min fill %d",
				ErrInvalidConfig, len(inner.children), Base)
		}
		if len(inner.children) > MaxChildren {
			return 0, 0, fmt.Errorf("%w: child count %d exceeds degree %d",
				ErrInvalidConfig, len(inner.children), MaxChildren)
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

func (t *Tree[I, S]) checkLeafInvariants(leaf *leafNode[I, S]) error {
	if leaf == nil {
		return fmt.Errorf("%w: nil leaf node", ErrInvalidConfig)
	}
	if int(leaf.n) != len(leaf.items) {
		return fmt.Errorf("%w: leaf occupancy mismatch (%d != %d)", ErrInvalidConfig, leaf.n, len(leaf.items))
	}
	if len(leaf.items) > len(leaf.itemStore) {
		return fmt.Errorf("%w: leaf len exceeds storage (%d > %d)", ErrInvalidConfig, len(leaf.items), len(leaf.itemStore))
	}
	if cap(leaf.items) != len(leaf.itemStore) {
		return fmt.Errorf("%w: leaf view cap mismatch (%d != %d)", ErrInvalidConfig, cap(leaf.items), len(leaf.itemStore))
	}
	if len(leaf.items) > 0 && &leaf.items[0] != &leaf.itemStore[0] {
		return fmt.Errorf("%w: leaf view is not backed by fixed storage", ErrInvalidConfig)
	}
	return nil
}

func (t *Tree[I, S]) checkInnerInvariants(inner *innerNode[I, S]) error {
	if inner == nil {
		return fmt.Errorf("%w: nil internal node", ErrInvalidConfig)
	}
	if int(inner.n) != len(inner.children) {
		return fmt.Errorf("%w: child occupancy mismatch (%d != %d)", ErrInvalidConfig, inner.n, len(inner.children))
	}
	if len(inner.children) > len(inner.childStore) {
		return fmt.Errorf("%w: child len exceeds storage (%d > %d)", ErrInvalidConfig, len(inner.children), len(inner.childStore))
	}
	if cap(inner.children) != len(inner.childStore) {
		return fmt.Errorf("%w: child view cap mismatch (%d != %d)", ErrInvalidConfig, cap(inner.children), len(inner.childStore))
	}
	if len(inner.children) > 0 && &inner.children[0] != &inner.childStore[0] {
		return fmt.Errorf("%w: child view is not backed by fixed storage", ErrInvalidConfig)
	}
	return nil
}
