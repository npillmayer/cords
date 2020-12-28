package cords

import (
	"bytes"
	"errors"
	"fmt"
)

// Cord is a type for an enhanced string.
type Cord struct {
	root *innerNode
}

func makeCord(node *cordNode) Cord {
	if node.IsLeaf() {
		r := makeInnerNode()
		r.attachLeft(node)
		return Cord{root: r}
	}
	// node is inner node
	inner := node.AsNode()
	if inner.right != nil {
		return Cord{root: inner}
	}
	r := makeInnerNode()
	r.attachLeft(&inner.cordNode)
	return Cord{root: r}
}

// FromString creates a cord from a Go string.
func FromString(s string) Cord {
	r := makeInnerNode()
	r.weight = uint64(len(s))
	r.height = 2 // leaf + inner node
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
func (cord Cord) each(f func(node *cordNode, depth int) error) error {
	err := traverse(&cord.root.cordNode, 0, f)
	return err
}

// EachLeaf iterates over all leaf nodes of the cord.
func (cord Cord) EachLeaf(f func(Leaf) error) error {
	var err error
	err = cord.each(func(node *cordNode, depth int) (e error) {
		if node.IsLeaf() {
			e = f(node.AsLeaf())
		}
		return
	})
	return err
}

// index locates the leaf containing index i.
func (cord Cord) index(i uint64) (*leafNode, uint64, error) {
	if cord.root == nil {
		return nil, 0, errIndexOutOfBounds
	}
	return index(&cord.root.cordNode, i)
}

// traverse walks a cord in in-order.
func traverse(node *cordNode, d int, f func(node *cordNode, depth int) error) error {
	if node.IsLeaf() {
		return f(node, d)
	}
	if l := node.AsNode().left; l != nil {
		if err := traverse(l, d+1, f); err != nil {
			return err
		}
	}
	if err := f(node, d); err != nil {
		return err
	}
	if r := node.AsNode().right; r != nil {
		if err := traverse(r, d+1, f); err != nil {
			return err
		}
	}
	return nil
}

var errIndexOutOfBounds error = errors.New("index out of bounds")

// index finds the leaf node of a cord which contains a given index.
// Return values are the leaf node, the index within the leaf, and a possible error.
func index(node *cordNode, i uint64) (*leafNode, uint64, error) {
	if node.Weight() <= i && node.Right() != nil {
		return index(node.Right(), i-node.Weight())
	}
	if node.Left() != nil {
		return index(node.Left(), i)
	}
	if i < uint64(node.Weight()) {
		if node.IsLeaf() {
			return node.AsLeaf(), i, nil
		}
		panic("index node is not a leaf")
	}
	return nil, i, errIndexOutOfBounds
}

// ---------------------------------------------------------------------------

// Leaf is an interface type for leafs of a cord structure.
// Leafs do carry string fragments.
type Leaf interface {
	Weight() uint64
	String() string
	Split(uint64) (Leaf, Leaf)
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

func (node *cordNode) Height() int {
	if node.IsLeaf() {
		return 1
	}
	n := node.AsNode()
	return n.height
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
	return fmt.Sprintf("<inner node |%d|, left=%v, right=%v>", node.Height(), node.Left(), node.Right())
}
func (inner *innerNode) attachLeft(child *cordNode) {
	inner.left = child
	child.parent = inner
	inner.adjustHeight()
}

func (inner *innerNode) attachRight(child *cordNode) {
	inner.right = child
	child.parent = inner
	inner.adjustHeight()
}

func (inner *innerNode) adjustHeight() int {
	mx := 0
	if inner.left != nil {
		mx = inner.left.Height()
		T().Debugf("|left| = %d", mx)
	}
	if inner.right != nil {
		h := inner.right.Height()
		T().Debugf("|right| = %d", h)
		mx = max(h, mx)
	}
	inner.height = mx + 1
	T().Debugf("setting height %d to %d", inner.height, mx+1)
	return mx + 1
}

func (leaf *leafNode) Weight() uint64 {
	return leaf.leaf.Weight()
}

func (leaf *leafNode) String() string {
	return leaf.leaf.String()
}

func (leaf *leafNode) Split(i uint64) (Leaf, Leaf) {
	l1, l2 := leaf.leaf.Split(i)
	return l1, l2
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

func (lstr leafString) Split(i uint64) (Leaf, Leaf) {
	left := lstr[:i]
	right := lstr[i:]
	return left, right
}

var _ Leaf = leafString("")

// ---------------------------------------------------------------------------

func dump(node *cordNode) {
	traverse(node, 0, func(node *cordNode, depth int) error {
		if node.IsLeaf() {
			l := node.AsLeaf()
			T().Debugf("%sL = %v", indent(depth), l)
			return nil
		}
		n := node.AsNode()
		T().Debugf("%sN = %v", indent(depth), n)
		return nil
	})
}

func indent(d int) string {
	ind := ""
	for d > 0 {
		ind = ind + "  "
		d--
	}
	return ind
}
