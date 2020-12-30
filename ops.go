package cords

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
	return cord
}

// Insert inserts a substring-cord c into cord at index i, resulting in a
// new cord. If i is greater than the length of cord, an out-of-bounds error
// is returned.
func Insert(cord Cord, c Cord, i uint64) (Cord, error) {
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
	return Concat(cl, c, cr), nil
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
	root := &clone(cord.root).cordNode
	node := root.Left()
	root2, err := cutRight(node, i, root, nil)
	if err != nil || root2 == nil {
		return cord, Cord{}, err
	}
	return Cord{root: root.AsInner()}, makeCord(root2), nil
}

// Delete removes a substring [i…i+l) from a cord, returning a new cord or an error.
// If j==0, cord is unchanged.
func Delete(cord Cord, i, l uint64) (Cord, error) {
	if l == 0 {
		return cord, nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return Cord{}, ErrIndexOutOfBounds
	}
	var c1, c2 Cord
	var err error
	if i > 0 {
		c1, c2, err = Split(cord, i)
		if err != nil {
			return cord, err
		}
	} else {
		c2 = cord
	}
	if i+l < cord.Len() {
		_, c2, err = Split(c2, l)
		if err != nil {
			return cord, err
		}
	}
	return Concat(c1, c2), nil
}

// Report outputs a substring: Report(i,l) ⇒ output the string bi,…,bi+l−1.
func (cord Cord) Report(i, l uint64) (string, error) {
	if l == 0 {
		return "", nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return "", ErrIndexOutOfBounds
	}
	var buf bytes.Buffer
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
	//root.weight = c1.Len() + c2.Len()
	node := cloneNode(c.root.left)
	c1root.attachRight(node)
	//dump(&c1root.cordNode)
	root.attachLeft(&c1root.cordNode)
	//
	cord = Cord{root: root} // new cord with new root to return
	if cord.Len() != cord.root.Len() {
		panic("structural inconsistency after concatentation")
	}
	if !cord.IsVoid() && unbalanced(cord.root.Left()) {
		b := balance(cord.root.Left().AsInner())
		cord.root.attachLeft(&b.cordNode)
	}
	if cord.Len() != cord.root.Len() {
		T().Debugf("cord.len=%d, cord.root.len=%d", cord.Len(), cord.root.Len())
		panic("structural inconsistency after re-balance")
	}
	return cord
}

func substr(node *cordNode, i, j uint64, buf bytes.Buffer) bytes.Buffer {
	T().Debugf("called substr([%d], %d, %d)", node.Weight(), i, j)
	if node.IsLeaf() {
		leaf := node.AsLeaf()
		T().Debugf("substr(%s|%d, %d, %d)", leaf, leaf.Len(), i, j)
		s := leaf.leaf.Substring(umax(0, i), umin(j, leaf.Len()))
		buf.WriteString(s)
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

func cutRight(node *cordNode, i uint64, parent *cordNode, root *cordNode) (*cordNode, error) {
	if node.Weight() <= i && node.Right() != nil { // node is inner node, walk right
		node = parent.swapNodeClone(node) // copy on write
		T().Debugf("split: traversing RIGHT")
		return cutRight(node.Right(), i-node.Weight(), node, root)
	}
	if node.Left() != nil { // node is inner node, may walk left
		if node.Weight() == i { // on mark ⇒ remove subtree starting at node.left, and done
			T().Debugf("split: clean cut of SUBTREE")
			root = concat(node, root) // cut off whole subtree starting at node
			parent.AsInner().right = nil
			return root, nil // no need to walk further down (left)
		}
		node = parent.swapNodeClone(node) // copy on write
		if node.Right() != nil {          // cut off right child
			root = concat(node.Right(), root)
			node.AsInner().right = nil
		}
		if parent.Right() == nil { // collapse node and parent (have identical metric)
			parent.AsInner().left = node.AsInner().left
			node = parent
			node.AsInner().height = node.Height() - 1
			T().Debugf("collapsing node with parent, height=%d", node.Height())
		}
		T().Debugf("split: traversing LEFT") // walk further down to the left
		return cutRight(node.Left(), i, node, root)
	}
	if i < uint64(node.Weight()) {
		T().Debugf("split: leaf split at %d in %v", i, node)
		if !node.IsLeaf() {
			panic("index node is not a leaf")
		}
		if i == 0 { // we must be in a right-side leaf
			root = concat(node, root)    // collect whole leaf
			parent.AsInner().right = nil // cut off whole leaf
		} else { // either left or right leaf, have to split it; leave parent intact
			l1, l2 := node.AsLeaf().split(i)
			if parent.Left() == node { // cut off l2 from left child
				// right sibling of leaf already cut off
				//parent.AsInner().left = &l1.cordNode
				T().Debugf("attaching left child with w=%d, parent.w=%d", l1.Weight(), parent.Weight())
				parent.AsInner().attachLeft(&l1.cordNode)
				T().Debugf("attaching left child with w=%d, parent.w=%d", l1.Weight(), parent.Weight())
			} else { // cut off l2 from right child
				//parent.AsInner().right = &l1.cordNode
				parent.AsInner().attachRight(&l1.cordNode)
			}
			root = concat(&l2.cordNode, root) // collect right part of split leaf
		}
		return root, nil // no going deeper
	}
	return nil, ErrIndexOutOfBounds
}

// ---------------------------------------------------------------------------

// traverse walks a cord in in-order.
func traverse(node *cordNode, d int, f func(node *cordNode, depth int) error) error {
	if node.IsLeaf() {
		return f(node, d)
	}
	if l := node.AsInner().left; l != nil {
		if err := traverse(l, d+1, f); err != nil {
			return err
		}
	}
	if err := f(node, d); err != nil {
		return err
	}
	if r := node.AsInner().right; r != nil {
		if err := traverse(r, d+1, f); err != nil {
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

func length(node *cordNode) uint64 {
	l := uint64(0)
	for node != nil {
		l += node.Weight()
		node = node.Right()
	}
	return l
}

func split(leaf *leafNode, i uint64) *innerNode {
	str := leaf.String()
	lstr := str[:i]
	rstr := str[i:]
	inner := makeInnerNode()
	inner.attachLeft(&makeStringLeaf(lstr).cordNode)
	inner.attachRight(&makeStringLeaf(rstr).cordNode)
	T().Debugf("after split, height of inner node = %d", inner.height)
	dump(&inner.cordNode)
	return inner
}

func balance(inner *innerNode) *innerNode {
	if inner.left == nil && inner.right == nil {
		return inner
	}
	if inner.left == nil {
		if inner.right.Height() > 2 {
			inner = rotateLeft(inner)
		} else {
			return inner
		}
	}
	if inner.right == nil {
		if inner.left.Height() > 2 {
			inner = rotateRight(inner)
		} else {
			return inner
		}
	}
	if inner.left == nil || inner.right == nil {
		panic("child is nil, should not be")
	}
	// now neither left nor right is nil
	for inner.right.Height() > inner.left.Height()+1 {
		inner = rotateLeft(inner)
	}
	for inner.left.Height() > inner.right.Height()+1 {
		inner = rotateRight(inner)
	}
	return inner
}

func rotateLeft(inner *innerNode) *innerNode {
	T().Debugf("rotate left")
	pivot := clone(inner.right.AsInner()) // clone pivot
	inner = clone(inner)                  // and inner: copy on write
	inner.attachRight(pivot.Left())       // inner.right = pivot.Left()
	pivot.attachLeft(&inner.cordNode)     // pivot.left = &inner.cordNode
	// inner.adjustHeight()
	// pivot.adjustHeight() // sequence matters
	return pivot
}

func rotateRight(inner *innerNode) *innerNode {
	T().Debugf("rotate right")
	dump(&inner.cordNode)
	pivot := clone(inner.left.AsInner()) // clone pivot
	inner = clone(inner)                 // and inner: copy on write
	inner.attachLeft(pivot.Right())      // inner.left = pivot.Right()
	pivot.attachRight(&inner.cordNode)   // pivot.right = &inner.cordNode
	T().Debugf("-----")
	dump(&pivot.cordNode)
	// inner.adjustHeight()
	// pivot.adjustHeight() // sequence matters
	T().Debugf("=====")
	return pivot
}

const balanceThres int = 1

func unbalanced(node *cordNode) bool {
	if node.IsLeaf() {
		return false
	}
	inner := node.AsInner()
	if inner.left == nil && inner.right == nil {
		return false
	}
	if inner.left == nil {
		return inner.right.Height() > balanceThres
	}
	if inner.right == nil {
		return inner.left.Height() > balanceThres
	}
	return abs(inner.left.Height()-inner.right.Height()) > balanceThres
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
