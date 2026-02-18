package btree

import "fmt"

func (t *Tree[I, S]) checkBackendLeafInvariants(leaf *leafNode[I, S]) error {
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

func (t *Tree[I, S]) checkBackendInnerInvariants(inner *innerNode[I, S]) error {
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
