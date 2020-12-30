package cords

import (
	"bytes"
	"fmt"
)

// Cord is a type for an enhanced string.
// It references fragments of text, which are considered immutable.
// Fragments will be shared between cords. Cords change in a concurrency-safe way,
// as every modifying operation on a cord will create a copy of changed parts of the cord.
//
// A cord created by
//
//     Cord{}
//
// is a valid object and behaves like the empty string.
//
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
	inner := node.AsInner()
	if inner.right == nil {
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
	r.left = &leaf.cordNode
	return Cord{root: r}
}

// String returns the cord as a Go string. This may be an expensive operation,
// as it will allocate a buffer for all the bytes of the cord and collect all
// fragments to a single continuous string. When working with large amounts of
// text, clients should probably avoid to call this.
// Instead they should jump to a position within the cord and report a
// substring or use an iterator.
func (cord Cord) String() string {
	if cord.IsVoid() {
		return ""
	}
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

func (cord Cord) height() int {
	if cord.IsVoid() {
		return 0
	}
	return cord.root.Height()
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
	return cord.root == nil || cord.root.Left() == nil || cord.Len() == 0
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
			e = f(node.AsLeaf().leaf)
		}
		return
	})
	return err
}

// index locates the leaf containing index i.
func (cord Cord) index(i uint64) (*leafNode, uint64, error) {
	if cord.root == nil {
		return nil, 0, ErrIndexOutOfBounds
	}
	return index(&cord.root.cordNode, i)
}

// ---------------------------------------------------------------------------

// Leaf is an interface type for leafs of a cord structure.
// Leafs do carry fragments of text.
// The default implementation uses Go strings.
type Leaf interface {
	Weight() uint64                  // length of the leaf fragment in bytes
	String() string                  // produce the leaf fragment as a string
	Substring(uint64, uint64) string // substring [i:j]
	Split(uint64) (Leaf, Leaf)       // split into 2 leafs at position i
}

// ---------------------------------------------------------------------------

type cordNode struct {
	self interface{}
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

func (node *cordNode) AsInner() *innerNode {
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
	n := node.AsInner()
	return n.weight
}

func (node *cordNode) Height() int {
	if node.IsLeaf() {
		return 1
	}
	n := node.AsInner()
	return n.height
}

func (node *cordNode) Len() uint64 {
	if node.IsLeaf() {
		return node.Weight()
	}
	inner := node.AsInner()
	l := uint64(inner.Weight())
	for inner.right != nil {
		l += inner.right.Weight()
		if inner.right.IsLeaf() {
			break
		}
		inner = inner.right.AsInner()
	}
	return l
}

func (node *cordNode) Left() *cordNode {
	if node.IsLeaf() {
		return nil
	}
	n := node.AsInner()
	return n.left
}

func (node *cordNode) Right() *cordNode {
	if node.IsLeaf() {
		return nil
	}
	n := node.AsInner()
	return n.right
}

func (node *cordNode) String() string {
	if node.IsLeaf() {
		return node.AsLeaf().String()
	}
	return fmt.Sprintf("<inner %d|%d|>", node.Weight(), node.Height())
	//return fmt.Sprintf("<inner %d|%d|, L=%v, R=%v>", node.Weight(), node.Height(), node.Left(), node.Right())
}

func (node *cordNode) swapNodeClone(child *cordNode) *cordNode {
	if node.IsLeaf() { // node must be an inner node
		panic("parent node is not of type inner node")
	}
	cln := cloneNode(child)
	inner := node.AsInner()
	if inner.left == child {
		inner.left = cln
	} else if inner.right == child {
		inner.right = cln
	} else {
		panic("node to clone is not a child of this parent")
	}
	return cln
}

func (inner *innerNode) attachLeft(child *cordNode) {
	inner.left = child
	inner.adjustHeight()
	if child != nil {
		inner.weight = child.Len()
	}
}

func (inner *innerNode) attachRight(child *cordNode) {
	inner.right = child
	inner.adjustHeight()
}

func (inner *innerNode) adjustHeight() int {
	mx := 0
	if inner.left != nil {
		mx = inner.left.Height()
		//T().Debugf("|left| = %d", mx)
	}
	if inner.right != nil {
		h := inner.right.Height()
		//T().Debugf("|right| = %d", h)
		mx = max(h, mx)
	}
	inner.height = mx + 1
	//T().Debugf("setting height %d to %d", inner.height, mx+1)
	return mx + 1
}

func (inner *innerNode) leftHeight() int {
	if inner.left == nil {
		return 0
	}
	return inner.left.Height()
}

func (inner *innerNode) rightHeight() int {
	if inner.right == nil {
		return 0
	}
	return inner.right.Height()
}

func (leaf *leafNode) Weight() uint64 {
	return leaf.leaf.Weight()
}

func (leaf *leafNode) String() string {
	return leaf.leaf.String()
}

func (leaf *leafNode) split(i uint64) (*leafNode, *leafNode) {
	l1, l2 := leaf.leaf.Split(i)
	ln1 := makeLeafNode()
	ln1.leaf = l1
	ln2 := makeLeafNode()
	ln2.leaf = l2
	return ln1, ln2
}

// --- Default Leaf implementation -------------------------------------------

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

func (lstr leafString) Substring(i, j uint64) string {
	return string(lstr)[i:j]
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
		n := node.AsInner()
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
