//go:build !btree_fixed

package btree

func (t *Tree[I, S]) checkBackendLeafInvariants(leaf *leafNode[I, S]) error {
	return nil
}

func (t *Tree[I, S]) checkBackendInnerInvariants(inner *innerNode[I, S]) error {
	return nil
}
