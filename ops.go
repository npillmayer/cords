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

// Concat concatenates cords and returns a new cord.
//
func Concat(cord Cord, others ...Cord) Cord {
	var nonvoid []Cord
	if !cord.IsVoid() {
		nonvoid = append(nonvoid, cord)
	}
	for _, c := range others {
		if !c.IsVoid() {
			nonvoid = append(nonvoid, c)
		}
	}
	if len(nonvoid) == 0 {
		return cord
	}
	if len(nonvoid) == 1 {
		return nonvoid[0]
	}
	cord = nonvoid[0]
	for _, c := range nonvoid[1:] {
		if c.Len() != c.root.Len() {
			panic(fmt.Sprintf("structural inconsistency, %d ≠ %d", c.Len(), c.root.Len()))
		}
		cord = cord.concat2(c)
	}
	// if !cord.IsVoid() && unbalanced(cord.root.Left()) {
	// 	panic("concat returns unbalanced cord")
	// }
	return cord
}

// Insert inserts a substring-cord c into cord at index i, resulting in a
// new cord. If i is greater than the length of cord, an out-of-bounds error
// is returned.
func Insert(cord Cord, c Cord, i uint64) (Cord, error) {
	dump(&cord.root.cordNode)
	if cord.IsVoid() && i == uint64(0) {
		return c, nil
	}
	if cord.Len() < i {
		return Cord{}, ErrIndexOutOfBounds
	}
	if c.IsVoid() {
		return cord, nil
	}
	if cord.Len() == i { // simply append at end
		return cord.concat2(c), nil
	}
	cl, cr, err := Split(cord, i)
	if err != nil {
		return cord, err
	}
	// ccc := Concat(cl, c, cr)
	// if ccc.root.Left() != nil && !ccc.root.Left().IsLeaf() {
	// 	ccc = makeCord(&balance(ccc.root.Left().AsInner()).cordNode)
	// }
	//return Concat(ccc), nil
	ccc := Concat(cl, c, cr)
	checkForRightNil(ccc)
	return ccc, nil
}

func checkForRightNil(c Cord) {
	traverse(c.root.Left(), c.root.Weight(), 0,
		func(node *cordNode, pos uint64, depth int) error {
			if !node.IsLeaf() && node.Right() == nil {
				T().Debugf("-----------------")
				dump(&c.root.cordNode)
				panic("check: right is nil")
			}
			return nil
		})
}

// Split splits a cord into two new (smaller) cords right before position i.
// Split(C,i) ⇒ split C into C1 and C2, with C1=b0,…,bi-1 and C2=bi,…,bn.
//
func Split(cord Cord, i uint64) (Cord, Cord, error) {
	if i == 0 {
		return Cord{}, cord, nil
	} else if i == cord.Len() {
		return cord, Cord{}, nil
	}
	if i > cord.Len() {
		return cord, Cord{}, ErrIndexOutOfBounds
	}
	if cord.root == nil || cord.root.Left() == nil {
		return cord, Cord{}, ErrIndexOutOfBounds
	}
	T().Debugf("........Split...........")
	dump(&cord.root.cordNode)
	root := &clone(cord.root).cordNode
	node := root.Left()
	root2, err := unzip(node, i, root, nil)
	if err != nil || root2 == nil {
		return cord, Cord{}, err
	}
	root.AsInner().left = tighten(root.Left())
	root.AsInner().weight = root.Left().Len()
	c1, c2 := makeCord(root), makeCord(root2)
	checkForRightNil(c1)
	checkForRightNil(c2)
	return balanceRoot(c1), balanceRoot(c2), nil
}

// Cut cuts out a substring [i…i+l) from a cord. It will return a new cord without
// the cut-out segment, plus the substring-segment, and possibly an error.
// If l is 0, cord is unchanged.
func Cut(cord Cord, i, l uint64) (Cord, Cord, error) {
	if l == 0 {
		return cord, Cord{}, nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return Cord{}, Cord{}, ErrIndexOutOfBounds
	}
	var c1, c2, c3 Cord
	var err error
	if i > 0 {
		c1, c2, err = Split(cord, i)
		if err != nil {
			return cord, Cord{}, err
		}
	} else {
		c2 = cord
	}
	if i+l < cord.Len() {
		c2, c3, err = Split(c2, l)
		if err != nil {
			return cord, Cord{}, err
		}
	}
	return Concat(c1, c3), c2, nil
}

// Report outputs a substring: Report(i,l) ⇒ outputs the string bi,…,bi+l−1.
func (cord Cord) Report(i, l uint64) (string, error) {
	if l == 0 {
		return "", nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return "", ErrIndexOutOfBounds
	}
	buf := new(bytes.Buffer)
	buf = substr(&cord.root.cordNode, i, i+l, buf)
	return buf.String(), nil
}

// Substr creates a new cord from a subset of cord, with:
// Substr(C,i,l) ⇒ Cord C2=bi,…,bi+l−1.
func Substr(cord Cord, i, l uint64) (Cord, error) {
	if l == 0 {
		return Cord{}, nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return cord, ErrIndexOutOfBounds
	}
	_, c, err := Split(cord, i)
	if i+l == cord.Len() {
		return c, err
	}
	c, _, err = Split(c, l)
	return c, err
}

// Index returns the cord Leaf (i.e., a text fragment) which includes the byte
// at position i, together with an index position within the fragment's text.
//
func (cord Cord) Index(i uint64) (Leaf, uint64, error) {
	if cord.Len() < i {
		return nil, 0, ErrIndexOutOfBounds
	}
	node, j, err := index(&cord.root.cordNode, i)
	if err != nil {
		return nil, 0, err
	}
	if !node.IsLeaf() {
		panic("cord.Index: node is not a leaf, but should be")
	}
	return node.AsLeaf().leaf, j, nil
}

// ---------------------------------------------------------------------------

// concat2 appends another cord to this cord, resulting in a new cord.
func (cord Cord) concat2(c Cord) Cord {
	if cord.IsVoid() {
		return c
	} else if c.IsVoid() {
		return cord
	}
	// we will set c2.root as the right child of clone(c1.root)
	c1root := clone(cord.root) // c1.root will change; copy on write
	root := makeInnerNode()    // root of new cord
	node := cloneNode(c.root.left)
	c1root.attachRight(node)
	root.attachLeft(&c1root.cordNode)
	//
	cord = Cord{root: root} // new cord with new root to return
	if cord.Len() != cord.root.Len() {
		T().Debugf("cord.len=%d, cord.root.len=%d", cord.Len(), cord.root.Len())
		panic("structural inconsistency after concatentation")
	}
	if !cord.IsVoid() && unbalanced(cord.root.Left()) {
		b := balance(cord.root.Left().AsInner())
		// if unbalanced(&b.cordNode) {
		// 	panic("new root is unbalanced after balancing")
		// }
		cord.root.attachLeft(&b.cordNode)
	}
	if cord.Len() != cord.root.Len() {
		T().Debugf("cord.len=%d, cord.root.len=%d", cord.Len(), cord.root.Len())
		panic("structural inconsistency after re-balance")
	}
	// if !cord.IsVoid() && unbalanced(cord.root.Left()) {
	// 	panic("concat2 returns unbalanced cord")
	// }
	return cord
}

func substr(node *cordNode, i, j uint64, buf *bytes.Buffer) *bytes.Buffer {
	T().Debugf("called substr([%d], %d, %d)", node.Weight(), i, j)
	if node.IsLeaf() {
		leaf := node.AsLeaf()
		T().Debugf("substr(%s|%d, %d, %d)", leaf, leaf.Len(), i, j)
		s := leaf.leaf.Substring(umax(0, i), umin(j, leaf.Len()))
		buf.Write(s)
		return buf
	}
	if i < node.Weight() && node.Left() != nil {
		buf = substr(node.Left(), i, j, buf)
	}
	if node.Right() != nil && j > node.Weight() {
		w := node.Weight()
		buf = substr(node.Right(), i-umin(w, i), j-w, buf)
	}
	//T().Debugf("node=%v", node)
	T().Debugf("dropping out of substr([%d], %d, %d)", node.Weight(), i, j)
	return buf
}

// unzip walks downward and cuts off all right children of nodes with weight > i.
// cut-off children are collected to a cord and returned.
// For better understanding, refer to Wikipedia: Rope (Data Structure),
// example for Split(…)
func unzip(node *cordNode, i uint64, parent *cordNode, root *cordNode) (*cordNode, error) {
	T().Debugf("unzip: i=%d, node=%v", i, node)
	if node.Weight() <= i && node.Right() != nil { // node is inner node, walk right
		//if node.Weight() < i && node.Right() != nil { // node is inner node, walk right
		node = parent.swapNodeClone(node) // copy on write
		T().Debugf("split: traversing RIGHT")
		return unzip(node.Right(), i-node.Weight(), node, root)
	}
	if node.Left() != nil { // node is inner node, may walk left
		if node.Weight() == i { // on mark ⇒ remove subtree starting at node.left, and done
			T().Debugf("split: clean cut of SUBTREE")
			root = concat(node, root) // cut off whole subtree starting at node
			parent.AsInner().attachRight(nil)
			return root, nil // no need to walk further down (left)
		}
		node = parent.swapNodeClone(node) // copy on write
		if node.Right() != nil {          // cut off right child
			root = concat(node.Right(), root)
			node.AsInner().attachRight(nil)
		}
		if parent.Right() == nil { // collapse node and parent (have identical metric)
			parent.AsInner().left = node.AsInner().left
			node = parent
			node.AsInner().height = node.Height() - 1
			T().Debugf("collapsing node with parent, height=%d", node.Height())
		}
		T().Debugf("split: traversing LEFT") // walk further down to the left
		return unzip(node.Left(), i, node, root)
	}
	// in leaf
	if i < uint64(node.Weight()) {
		T().Debugf("split: leaf split at %d in %v", i, node)
		if !node.IsLeaf() {
			panic("index node is not a leaf")
		}
		if i == 0 { // we must be in a right-side leaf
			root = concat(node, root)         // collect whole leaf
			parent.AsInner().attachRight(nil) // cut off whole leaf
			T().Debugf("clean cut of rhs leaf")
		} else { // either left or right leaf, have to split it; leave parent intact
			l1, l2 := node.AsLeaf().split(i)
			if parent.Left() == node { // cut off l2 from left child
				// right sibling of leaf already cut off
				T().Debugf("attaching left child with w=%d, parent.w=%d", l1.Weight(), parent.Weight())
				parent.AsInner().attachLeft(&l1.cordNode)
				T().Debugf("attaching left child with w=%d, parent.w=%d", l1.Weight(), parent.Weight())
			} else { // cut off l2 from right child
				parent.AsInner().attachRight(&l1.cordNode)
			}
			root = concat(&l2.cordNode, root) // collect right part of split leaf
		}
		return root, nil // no going deeper
	}
	return nil, ErrIndexOutOfBounds
}

// tighten walks down the cut line after a split. Due to the cut off right
// children, weights may be out of sync. We walk downwards recursively
// until we reach the most rightward leaf, then back upwards, setting the
// weights to the correct values.
func tighten(node *cordNode) *cordNode {
	T().Debugf("tighten %v", node)
	if node == nil || node.IsLeaf() {
		return node
	}
	if node.Right() != nil { // keep weight unchanged
		node.AsInner().right = tighten(node.Right())
		//tighten(node.Right())
		node.AsInner().adjustHeight()
		return node
	}
	// node.right == nil ⇒ we are on the cut line
	if node.Left() == nil { // node has no leafs, impossible
		panic("Inner node without leaf")
	}
	if node.Right() != nil {
		panic("node.right != nil")
	}
	//
	left := tighten(node.Left())
	if left.IsLeaf() {
		//if node.Right() == nil {
		//panic(fmt.Sprintf("left is leaf %v, right is nil", left))
		return left
		//}
		// node.AsInner().left = left
		// node.AsInner().weight = left.Weight()
		// return node
	}
	if left.Right() == nil { // collapse child with node
		panic("left.right is nil")
		//node.AsInner().attachLeft(left.Left())
	}
	return left
	// node.AsInner().left = left
	// node.AsInner().weight = left.Len()
	// node.AsInner().adjustHeight()
	// return node
}

// ---------------------------------------------------------------------------

// traverse walks a cord in in-order.
func traverse(node *cordNode, pos uint64, depth int,
	f func(node *cordNode, pos uint64, depth int) error) error {
	//
	if node.IsLeaf() {
		return f(node, pos-node.Weight(), depth)
	}
	inner := node.AsInner()
	if l := inner.left; l != nil {
		leftpos := pos - inner.weight + l.Weight()
		if err := traverse(l, leftpos, depth+1, f); err != nil {
			return err
		}
	}
	if err := f(node, pos, depth); err != nil {
		return err
	}
	if r := inner.right; r != nil {
		rightpos := pos + r.Weight()
		if err := traverse(r, rightpos, depth+1, f); err != nil {
			return err
		}
	}
	return nil
}

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
	return nil, i, ErrIndexOutOfBounds
}

func concat(n1, n2 *cordNode) *cordNode {
	if n1 == nil {
		return cloneNode(n2)
	} else if n2 == nil {
		return cloneNode(n1)
	}
	inner := makeInnerNode()
	inner.attachLeft(cloneNode(n1))
	inner.attachRight(n2)
	return &inner.cordNode
}

// length calculates the string length of a subtree by adding the weight of
// the node, plus the weight of the right child, its child, its child etc.
// down to the rightmost leaf.
func length(node *cordNode) uint64 {
	l := uint64(0)
	for node != nil {
		l += node.Weight()
		node = node.Right()
	}
	return l
}

// split splits a leaf node at i, creates a new inner node and attaches the
// new leaves to it.
func split(leaf *leafNode, i uint64) *innerNode {
	str := leaf.String()
	lstr := str[:i]
	rstr := str[i:]
	inner := makeInnerNode()
	inner.attachLeft(&makeStringLeafNode(lstr).cordNode)
	inner.attachRight(&makeStringLeafNode(rstr).cordNode)
	//dump(&inner.cordNode)
	return inner
}

// balance recursively balances unbalanced subtrees, using right- and left-rotations.
func balance(inner *innerNode) *innerNode {
	if inner.left == nil && inner.right == nil {
		return inner
	}
	//dump(&inner.cordNode)
	//T().Debugf("-----------")
	if unbalanced(inner.left) {
		x := balance(inner.left.AsInner())
		inner.attachLeft(&x.cordNode)
		// if unbalanced(inner.left) {
		// 	panic("inner.left is unbalanced after balancing")
		// }
	}
	if unbalanced(inner.right) {
		x := balance(inner.right.AsInner())
		inner.attachRight(&x.cordNode)
		// if unbalanced(inner.right) {
		// 	panic("inner.right is unbalanced after balancing")
		// }
	}
	cnt := 0
	for cnt < 10 && unbalanced(&inner.cordNode) {
		if inner.rightHeight() > inner.leftHeight()+balanceThres {
			inner = rotateLeft(inner)
		}
		if inner.leftHeight() > inner.rightHeight()+balanceThres {
			inner = rotateRight(inner)
		}
		if unbalanced(inner.Left()) {
			inner.left = &balance(inner.left.AsInner()).cordNode
		}
		if unbalanced(inner.Right()) {
			inner.right = &balance(inner.right.AsInner()).cordNode
		}
		cnt++
	}
	// if unbalanced(&inner.cordNode) {
	// 	dump(&inner.cordNode)
	// 	panic("inner is unbalanced")
	// }
	return inner
}

func balanceRoot(c Cord) Cord {
	if c.IsVoid() || c.root.left.IsLeaf() {
		return c
	}
	c.root.left = &balance(c.root.left.AsInner()).cordNode
	return c
}

// rotatedLeft performs a left rotation on an inner tree node.
// Wikipedia has a good article on tree rotation.
func rotateLeft(inner *innerNode) *innerNode {
	//T().Debugf("rotate left")
	pivot := clone(inner.right.AsInner()) // clone pivot
	inner = clone(inner)                  // and inner: copy on write
	inner.attachRight(pivot.Left())       // inner.right = pivot.Left()
	pivot.attachLeft(&inner.cordNode)     // pivot.left = &inner.cordNode
	return pivot
}

// rotatedLeft performs a right rotation on an inner tree node.
// Wikipedia has a good article on tree rotation.
func rotateRight(inner *innerNode) *innerNode {
	//T().Debugf("rotate right")
	pivot := clone(inner.left.AsInner()) // clone pivot
	inner = clone(inner)                 // and inner: copy on write
	inner.attachLeft(pivot.Right())      // inner.left = pivot.Right()
	pivot.attachRight(&inner.cordNode)   // pivot.right = &inner.cordNode
	return pivot
}

// we balance a subtree whenever the height-diff between left and right child
// is greater than this threshold.
const balanceThres int = 1

func unbalanced(node *cordNode) bool {
	if node == nil || node.IsLeaf() {
		return false
	}
	inner := node.AsInner()
	if inner.left == nil && inner.right == nil {
		panic("node without children")
	}
	if unbalanced(inner.left) || unbalanced(inner.right) {
		return true
	}
	return abs(inner.leftHeight()-inner.rightHeight()) > balanceThres
}

func clone(inner *innerNode) *innerNode {
	n := makeInnerNode()
	n.height = inner.height
	n.weight = inner.weight
	n.left = inner.left
	n.right = inner.right
	return n
}

func cloneLeaf(leaf *leafNode) *leafNode {
	l := makeLeafNode()
	l.leaf = leaf.leaf
	return l
}

func cloneNode(node *cordNode) *cordNode {
	if node == nil {
		return nil
	}
	if node.IsLeaf() {
		return &cloneLeaf(node.AsLeaf()).cordNode
	}
	return &clone(node.AsInner()).cordNode
}

// --- Helpers ---------------------------------------------------------------

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func umin(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func umax(a, b uint64) uint64 {
	if a < b {
		return b
	}
	return a
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
