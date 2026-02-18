//go:build !btree_fixed

package btree

type treeNode[I SummarizedItem[S], S any] interface {
	isLeaf() bool
	Summary() S
}

type leafNode[I SummarizedItem[S], S any] struct {
	summary S
	items   []I
}

func (l *leafNode[I, S]) isLeaf() bool { return true }
func (l *leafNode[I, S]) Summary() S   { return l.summary }

type innerNode[I SummarizedItem[S], S any] struct {
	summary  S
	children []treeNode[I, S]
}

func (n *innerNode[I, S]) isLeaf() bool { return false }
func (n *innerNode[I, S]) Summary() S   { return n.summary }
