package btree

import "fmt"

const (
	// Fixed storage capacities aligned with a TREE_BASE=6 shape.
	Base            = 6
	MaxChildren     = 2 * Base
	MaxLeafItems    = 2 * Base
	OverflowStorage = MaxChildren + 1 // transient overflow before split
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
	itemStore [OverflowStorage]I
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
	childStore [OverflowStorage]treeNode[I, S]
	// children is a dynamic-length view over childStore and must satisfy:
	// len(children) == int(n), cap(children) == len(childStore), children backed by childStore.
	children []treeNode[I, S]
}

func (n *innerNode[I, S]) isLeaf() bool { return false }
func (n *innerNode[I, S]) Summary() S   { return n.summary }

// SummarizedItem ties a leaf item to its summary type at compile time.
type SummarizedItem[S any] interface {
	Summary() S
}

// SummaryMonoid defines how summaries are aggregated up the tree.
//
// For summaries s, t, u, Add should be associative:
//
//	Add(Add(s, t), u) == Add(s, Add(t, u))
//
// and Zero should be the neutral element:
//
//	Add(Zero(), s) == s == Add(s, Zero())
type SummaryMonoid[S any] interface {
	Zero() S
	Add(left, right S) S
}

// Config configures a rope-focused B+ sum-tree.
type Config[S any] struct {
	// Monoid aggregates summaries up the tree.
	Monoid SummaryMonoid[S]
}

func (cfg Config[S]) normalized() Config[S] {
	return cfg
}

func (cfg Config[S]) validate() error {
	cfg = cfg.normalized()
	if cfg.Monoid == nil {
		return fmt.Errorf("%w: monoid is required", ErrInvalidConfig)
	}
	return nil
}
