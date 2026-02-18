package btree

const (
	// Fixed storage capacities aligned with a TREE_BASE=6 shape.
	fixedBase            = 6
	fixedMaxChildren     = 2 * fixedBase
	fixedMaxLeafItems    = 2 * fixedBase
	fixedOverflowStorage = fixedMaxChildren + 1 // transient overflow before split
)

type treeNode[I SummarizedItem[S], S any] interface {
	isLeaf() bool
	Summary() S
}

type leafNode[I SummarizedItem[S], S any] struct {
	summary S
	// n is the logical item count; valid items are itemStore[:n].
	n uint8
	// itemStore is the fixed backing storage for leaf items.
	itemStore [fixedOverflowStorage]I
	// items is a dynamic-length view over itemStore and must satisfy:
	// len(items) == int(n), cap(items) == len(itemStore), items backed by itemStore.
	items []I
}

func (l *leafNode[I, S]) isLeaf() bool { return true }
func (l *leafNode[I, S]) Summary() S   { return l.summary }

type innerNode[I SummarizedItem[S], S any] struct {
	summary S
	// n is the logical child count; valid children are childStore[:n].
	n uint8
	// childStore is the fixed backing storage for child pointers.
	childStore [fixedOverflowStorage]treeNode[I, S]
	// children is a dynamic-length view over childStore and must satisfy:
	// len(children) == int(n), cap(children) == len(childStore), children backed by childStore.
	children []treeNode[I, S]
}

func (n *innerNode[I, S]) isLeaf() bool { return false }
func (n *innerNode[I, S]) Summary() S   { return n.summary }
