package cords

/*
BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"bytes"
	"fmt"
)

// This implementation follows more or less the description of the `Rope´ data
// structure as described in Wikipedia. I recommend opening this page alongside
// the code for easier understanding.
//
// A cord builds a binary tree structure on top of string fragments. The root node
// of the tree is carried by a `Cord´ struct type. Inner nodes of the tree carry
// one or two children, a weight, and a height indicator. Some invariants hold:
//
//   * The height of a node is the maximum between left and right child's height.
//   * The weight of a node is equal to the total string length of the *left* subtree.
//   * The weight of a leaf is equal to the length of the string fragment it carries.
//   * The total string length of a subtree, starting from node N, is equal to N's
//     weight, plus the weight's of all the straight line of right children down to
//     the rightmost child of the subtree.
//   * The root node of a cord is either nil or has exactly a left child, no right one.
//   * No inner node without at least one child exists.

// Cord is a type for an enhanced string.
// It references fragments of text, which are considered immutable.
// Fragments will be shared between cords. Cords change in a concurrency-safe way,
// as every modifying operation on a cord will create a copy of changed parts of the cord,
// thus cords are persistent data structures.
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

// FromString creates a cord from a Go string.
func FromString(s string) Cord {
	r := makeInnerNode()
	r.weight = uint64(len(s))
	r.height = 2 // leaf + inner node
	leaf := makeStringLeafNode(s)
	r.left = &leaf.cordNode
	return Cord{root: r}
}

// makeCord is an internal helper to create a cord from a given cordNode, which
// shall be made the root of a new cord. Sometimes we are not quite sure of
// what type of node an operation yields. As the node structure of a cord
// has to follow some invariants, we use this function to always end up with
// a correct cord.
//
// A cord may be void, i.e., reflect the empty string. The root node may be
// nil in this case. Cord{} is a valid cord, reflecting "" (empty string).
//
// The root node of a non-void cord is always of type innerNode and has exactly one
// child, which is on its left. This way, the weight of the root node always will
// reflect the byte-length of the cord.
//
func makeCord(node *cordNode) Cord {
	if node.IsLeaf() {
		r := makeInnerNode()
		r.attachLeft(node)
		return Cord{root: r}
	}
	// given node is inner node
	inner := node.AsInner()
	if inner.right == nil { // we can use it directly
		return Cord{root: inner}
	}
	r := makeInnerNode() // otherwise we create a root node on top
	r.attachLeft(&inner.cordNode)
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

// IsVoid returns true if cord is "".
func (cord Cord) IsVoid() bool {
	return cord.root == nil || cord.root.Left() == nil || cord.Len() == 0
}

// Len returns the length in bytes of a cord.
func (cord Cord) Len() uint64 {
	if cord.root == nil {
		return 0
	}
	return cord.root.Weight()
}

// height returns the total height of a cords tree.
func (cord Cord) height() int {
	if cord.IsVoid() {
		return 0
	}
	return cord.root.Height()
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

// index locates the leaf containing index i. May return an out-of-bounds error.
// If successful, will return a reference to a leaf node and the position
// within the node.
func (cord Cord) index(i uint64) (*leafNode, uint64, error) {
	if cord.root == nil {
		return nil, 0, ErrIndexOutOfBounds
	}
	return index(&cord.root.cordNode, i)
}

// ---------------------------------------------------------------------------

// Leaf is an interface type for leafs of a cord structure.
// Leafs do carry fragments of text.
//
// The default implementation uses Go strings.
type Leaf interface {
	Weight() uint64                  // length of the leaf fragment in bytes
	String() string                  // produce the leaf fragment as a string
	Substring(uint64, uint64) []byte // substring [i:j]
	Split(uint64) (Leaf, Leaf)       // split into 2 leafs at position i
}

// --- Node types ------------------------------------------------------------

// We use 2 types of distinct nodes: inner nodes and leaf nodes.
// Inner nodes may have one or two children nodes. Leaf nodes point to a Leaf (interface).
// We define an interface type cordNode to unify some node operations. Every node
// will carry a reference to itself, so that node operations are able to
// distinguish the type of node they operate on. To ensure the 'self' reference
// to always be correctly initialized, we create nodes exclusively through the
// make…() methods below.
//
// One design decision is to not include a reference to the parent node. This is a
// trade-off which makes some algorithms a bit more cumbersome. On the other hand,
// this is necessary to be able to re-use subtrees and having a persistent
// (immutable) data structure without having to always clone the complete tree.
// Tree operations will clone certain nodes of a tree on modifications, but
// leave unchanged parts of the tree in place and rather reference them.

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

// swapNodeClone creates a clone from a node, which must be a child node
// of node. The newly created clone is then inserted in place of the child.
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

// attachLeft attaches a node as the left child of an inner node.
// Height and weight are adjusted. Adjusting the weight is an O(log n) operation.
func (inner *innerNode) attachLeft(child *cordNode) {
	inner.left = child
	inner.adjustHeight()
	if child != nil {
		inner.weight = child.Len()
	}
}

// attachLeft attaches a node as the right child of an inner node.
// Height is adjusted.
func (inner *innerNode) attachRight(child *cordNode) {
	inner.right = child
	inner.adjustHeight()
}

// adjustHeight sets the height of a node to max(left.H,right.H)+1.
func (inner *innerNode) adjustHeight() int {
	inner.height = max(inner.leftHeight(), inner.rightHeight()) + 1
	return inner.height
}

// leftHeight returns the height of the left child or 0.
func (inner *innerNode) leftHeight() int {
	if inner.left == nil {
		return 0
	}
	return inner.left.Height()
}

// rightHeight returns the height of the right child or 0.
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

// spit splits a leaf node at position i, resulting in 2 new leaf nodes.
// Interface Leaf must support the Split(…) operation.
func (leaf *leafNode) split(i uint64) (*leafNode, *leafNode) {
	l1, l2 := leaf.leaf.Split(i)
	ln1 := makeLeafNode()
	ln1.leaf = l1
	ln2 := makeLeafNode()
	ln2.leaf = l2
	return ln1, ln2
}

// --- Default Leaf implementation -------------------------------------------

//StringLeaf is the default implementation of type Leaf.
type StringLeaf string

// makeStringLeafNode creates a leaf node and a leaf from a given string.
func makeStringLeafNode(s string) *leafNode {
	leaf := makeLeafNode()
	leaf.leaf = StringLeaf(s)
	return leaf
}

// Weight of a leaf is its string length in bytes.
func (lstr StringLeaf) Weight() uint64 {
	return uint64(len(lstr))
}

func (lstr StringLeaf) String() string {
	return string(lstr)
}

// Split splits a leaf at position i, resulting in 2 new leafs.
func (lstr StringLeaf) Split(i uint64) (Leaf, Leaf) {
	left := lstr[:i]
	right := lstr[i:]
	return left, right
}

// Substring returns a string segment of the leaf's text fragment.
func (lstr StringLeaf) Substring(i, j uint64) []byte {
	return []byte(lstr)[i:j]
}

var _ Leaf = StringLeaf("")

// --- Debugging helper ------------------------------------------------------

func dump(node *cordNode) {
	traverse(node, 0, func(node *cordNode, depth int) error {
		if node.IsLeaf() {
			l := node.AsLeaf()
			T().Debugf("%sL = %v", indent(depth), strstart(l))
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

func strstart(leaf *leafNode) string {
	s := leaf.String()
	if len(s) > 8 {
		return s[:7] + "…"
	}
	return s
}
