package cords

import (
	"bytes"
	"errors"
)

// Cord is a type for an enhanced string.
type Cord struct {
	root *innerNode
}

// FromString creates a cord from a Go string.
func FromString(s string) Cord {
	r := makeInnerNode()
	r.weight = uint64(len(s))
	r.height = 1
	leaf := makeStringLeaf(s)
	leaf.parent = r
	r.left = &leaf.cordNode
	return Cord{root: r}
}

func (cord Cord) String() string {
	var bf bytes.Buffer
	var err error
	err = cord.EachLeaf(func(leaf Leaf) error {
		if _, err = bf.WriteString(leaf.String()); err != nil {
			T().Errorf(err.Error())
			return err
		}
		return nil
	})
	if err != nil {
		// TODO: what to do? String() should not return an error. Can there be an error?
	}
	return bf.String()
}

// Len returns the length in bytes of a cord.
func (cord Cord) Len() uint64 {
	if cord.root == nil {
		return 0
	}
	return cord.root.Weight()
}

// IsVoid returns true if cord is "".
func (cord Cord) IsVoid() bool {
	return cord.root == nil
}

// each iterates over all nodes of the cord.
func (cord Cord) each(f func(node *cordNode) error) error {
	err := traverse(&cord.root.cordNode, f)
	return err
}

// EachLeaf iterates over all leaf nodes of the cord.
func (cord Cord) EachLeaf(f func(Leaf) error) error {
	var err error
	err = cord.each(func(node *cordNode) (e error) {
		if node.IsLeaf() {
			e = f(node.AsLeaf())
		}
		return
	})
	return err
}

// traverse walks a cord in in-order.
func traverse(node *cordNode, f func(node *cordNode) error) error {
	if node.IsLeaf() {
		return f(node)
	}
	if l := node.AsNode().left; l != nil {
		if err := traverse(l, f); err != nil {
			return err
		}
	}
	if err := f(node); err != nil {
		return err
	}
	if r := node.AsNode().right; r != nil {
		if err := traverse(r, f); err != nil {
			return err
		}
	}
	return nil
}

var errIndexOutOfBounds error = errors.New("index out of bounds")

func (cord Cord) index(i uint64) (*cordNode, uint64, error) {
	if cord.root == nil {
		return nil, 0, errIndexOutOfBounds
	}
	return index(&cord.root.cordNode, i)
}

// index finds the leaf node of a cord which contains a given index.
// Return values are the leaf node, the index within the leaf, and a possible error.
func index(node *cordNode, i uint64) (*cordNode, uint64, error) {
	if node.Weight() <= i && node.Right() != nil {
		return index(node.Right(), i-node.Weight())
	}
	if node.Left() != nil {
		return index(node.Left(), i)
	}
	if i < uint64(node.Weight()) {
		return node, i, nil
	}
	return nil, i, errIndexOutOfBounds
}

func (cord Cord) balance() {
}

// ---------------------------------------------------------------------------

// Leaf is an interface type for leafs of a cord structure.
// Leafs do carry string fragments.
type Leaf interface {
	// Left() Leaf
	// Right() Leaf
	//Parent() Leaf
	Weight() uint64
	String() string
}

// ---------------------------------------------------------------------------

type cordNode struct {
	parent *innerNode
	self   interface{}
}

type innerNode struct {
	cordNode
	left, right *cordNode
	weight      uint64
	height      int
}

type leafNode struct {
	cordNode
	leaf Leaf
}

func makeInnerNode() *innerNode {
	inner := &innerNode{}
	inner.self = inner
	return inner
}

func makeLeafNode() *leafNode {
	leaf := &leafNode{}
	leaf.self = leaf
	return leaf
}

func (node *cordNode) AsNode() *innerNode {
	return node.self.(*innerNode)
}

func (node *cordNode) AsLeaf() *leafNode {
	return node.self.(*leafNode)
}

func (node *cordNode) IsLeaf() bool {
	if node.self == nil {
		panic("node has no self, inconsistency")
	}
	_, ok := node.self.(*leafNode)
	return ok
}

func (node *cordNode) Weight() uint64 {
	if node.IsLeaf() {
		return node.AsLeaf().Weight()
	}
	n := node.AsNode()
	return n.weight
}

func (node *cordNode) Left() *cordNode {
	if node.IsLeaf() {
		return nil
	}
	n := node.AsNode()
	return n.left
}

func (node *cordNode) Right() *cordNode {
	if node.IsLeaf() {
		return nil
	}
	n := node.AsNode()
	return n.right
}

func (node *cordNode) String() string {
	if node.IsLeaf() {
		return node.AsLeaf().String()
	}
	return "<inner node>"
}

func (leaf *leafNode) Weight() uint64 {
	return leaf.leaf.Weight()
}

func (leaf *leafNode) String() string {
	return leaf.leaf.String()
}

var _ Leaf = &leafNode{}

// ---------------------------------------------------------------------------

type leafString string

func makeStringLeaf(s string) *leafNode {
	leaf := makeLeafNode()
	leaf.leaf = leafString(s)
	return leaf
}

func (lstr leafString) Weight() uint64 {
	return uint64(len(lstr))
}

func (lstr leafString) String() string {
	return string(lstr)
}

var _ Leaf = leafString("")

// func (leaf stringLeaf) Left() Leaf {
// 	return nil
// }

// func (leaf stringLeaf) Right() Leaf {
// 	return nil
// }

// func (leaf stringLeaf) Parent() Leaf {
// 	return leaf.parent
// }
