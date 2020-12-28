package cords

// Concat concatenates two cords to a new one.
// The cord tree is balanced afterwards.
func Concat(c1, c2 Cord) Cord {
	if c1.IsVoid() {
		return c2
	}
	if c2.IsVoid() {
		return c1
	}
	// we will set c2.root as the right child of clone(c1.root)
	c1root := clone(c1.root) // c1.root will change; copy on write
	root := makeInnerNode()  // root of new cord
	root.weight = c1.Len() + c2.Len()
	node := cloneNode(c2.root.left)
	c1root.attachRight(node)
	root.attachLeft(&c1root.cordNode)
	//root.height = max(c1.root.height, c2.root.height) + 1 // done by attach…(…)
	//
	cord := Cord{root: root} // new cord with new root to return
	if unbalanced(cord.root) {
		cord.root = balance(cord.root)
	}
	return cord
}

func Split(c Cord, i uint64) (Cord, Cord, error) {
	if c.root == nil || c.root.Left() == nil {
		return c, Cord{}, errIndexOutOfBounds
	}
	root := &clone(c.root).cordNode
	node := root.Left()
	root2, err := cutRight(node, i, root, nil)
	if err != nil || root2 == nil {
		return c, Cord{}, err
	}
	return Cord{root: root.AsNode()}, makeCord(root2), nil
}

// TODO: If right child is cut off, check if parent's parent has a right child.
// If not, move parent upwards (unify 2 nodes).
func cutRight(node *cordNode, i uint64, parent *cordNode, root *cordNode) (*cordNode, error) {
	if node.Weight() <= i && node.Right() != nil { // node is inner node, walk right
		// node = cloneNode(node)       // copy on write
		// parent.AsNode().right = node // parent is alredy cloned
		node = parent.swapNodeClone(node) // copy on write
		T().Debugf("split: traversing RIGHT")
		return cutRight(node.Right(), i-node.Weight(), node, root)
	}
	if node.Left() != nil { // node is inner node, may walk left
		if node.Weight() == i { // on mark ⇒ remove subtree starting at node.left, and done
			T().Debugf("split: clean cut of SUBTREE")
			root = concat(node, root) // cut off whole subtree starting at node
			// node = cloneNode(node)      // copy on write
			// parent.AsNode().right = nil // parent is alredy cloned
			//node = parent.swapNodeClone(node) // copy on write
			parent.AsNode().right = nil
			return root, nil // no need to walk further down (left)
		}
		node = parent.swapNodeClone(node) // copy on write
		if node.Right() != nil {          // cut off right child
			root = concat(node.Right(), root)
			node.AsNode().right = nil
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
			root = concat(node, root)
			parent.AsNode().right = nil
		} else { // either left or right leaf, have to split it
			l1, l2 := node.AsLeaf().split(i)
			if parent.Left() == node {
				// cut off l2
				// leave parent intact
				// right sibling of leaf already cut off
				parent.AsNode().left = &l1.cordNode
				root = concat(&l2.cordNode, root)
			} else {
				// cut off l2
				parent.AsNode().right = &l1.cordNode
				root = concat(&l2.cordNode, root)
			}
		}
		return root, nil
	}
	return nil, errIndexOutOfBounds
}

// func XSplit(c Cord, i uint64) (Cord, Cord, error) {
// 	// TODO case i == 0 or i == len(cord)
// 	dump(&c.root.cordNode)
// 	T().Debugf("----Split----------")
// 	leaf, inx, err := c.index(i)
// 	if err != nil {
// 		return c, Cord{}, err
// 	}
// 	dump(&c.root.cordNode)
// 	parent := leaf.parent
// 	x := parent
// 	T().Debugf("before split, parent = %v", parent)
// 	//var sibling *leafNode
// 	//if inx < leaf.Weight()-1 { // split mid-string
// 	if inx > 0 { // split mid-string
// 		p := split(leaf, inx)        // p is parent of new leaf and its left sibling
// 		p.parent = parent            // hook it up to orig leaf's parent
// 		parent = p                   // now call p the new leaf's parent
// 		leaf = parent.right.AsLeaf() // and call new leaf the leaf
// 		//sibling = parent.left.AsLeaf() // must have a left sibling
// 	}
// 	// else if lft := parent.Left(); lft != nil {
// 	// 	if sibling = lft.AsLeaf(); sibling == leaf {
// 	// 		sibling = nil
// 	// 	}
// 	// }
// 	//
// 	T().Debugf("after split, parent = %v", parent)
// 	T().Debugf("after split,      x = %v", x)
// 	// leaf and parent may not be connected
// 	// now leaf and parent are set ⇒ split it before leaf
// 	//root1 := unzip(leaf, c.root)   // produce a copy of all upward nodes until new left root
// 	root1 := unzip(&parent.cordNode, c.root) // produce a copy of all upward nodes until new left root
// 	dump(&root1.cordNode)
// 	panic("worked?")
// 	var root2 *cordNode
// 	// leaf may be left or right child of parent
// 	n := &leaf.cordNode
// 	// search upwards until n is right child of its parent => split point
// 	for n != &root1.cordNode { // walk upwards upto root
// 		if n.parent.right == n { // yes, n is right child
// 			break // stop walking
// 		}
// 		n = &n.parent.cordNode
// 	}
// 	p := n.parent
// 	p.right = nil  // split off n
// 	n.parent = nil // split off n
// 	root2 = n      // build up root2, starting with n
// 	weight := n.Weight()
// 	for p != nil { // walk upwards again, searching for right children
// 		p.weight -= weight
// 		if p.right != nil {
// 			weight += length(p.right)
// 			rt := clone(p.right.AsNode())
// 			rt.parent = nil
// 			p.right = nil
// 			root2 = concat(root2, &rt.cordNode)
// 		}
// 		p = p.parent
// 	}
// 	// TODO correctly pack roots into Cords
// 	return Cord{root: root1}, makeCord(root2), nil
// }

// ---------------------------------------------------------------------------

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

// func unzip(node *cordNode, stopper *innerNode) *innerNode {
// 	if node == nil {
// 		return nil
// 	}
// 	var inner *innerNode
// 	if node.IsLeaf() {
// 		node = &cloneLeaf(node.AsLeaf()).cordNode
// 		inner = node.parent
// 	} else {
// 		inner = node.AsNode()
// 	}
// 	T().Debugf("unzip start = %v", inner)
// 	for inner != nil && inner != stopper {
// 		inner = clone(inner)
// 		inner.adjustHeight()
// 		T().Debugf("unzip node = %v/%d", inner, inner.height)
// 		if inner.parent == nil || inner == stopper {
// 			T().Debugf("stopping with %v", inner)
// 			T().Debugf("   ⇒ parent is %v", inner.parent)
// 			break
// 		}
// 		inner = inner.parent
// 	}
// 	return inner
// }

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
		if inner.right.Height() > 1 {
			inner = rotateLeft(inner)
		} else {
			return inner
		}
	}
	if inner.right == nil {
		if inner.left.Height() > 1 {
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
	pivot := clone(inner.right.AsNode()) // clone pivot and inner; copy on write
	inner = clone(inner)
	inner.right = pivot.Left()
	inner.height = inner.Right().Height() + 1
	//pivot.Left().parent = inner
	pivot.left = &inner.cordNode
	//inner.parent = pivot
	//pivot.parent = nil
	inner.adjustHeight()
	pivot.adjustHeight() // sequence matters
	return pivot
}

func rotateRight(inner *innerNode) *innerNode {
	T().Debugf("rotate right")
	pivot := clone(inner.left.AsNode()) // clone pivot and inner; copy on write
	inner = clone(inner)
	inner.left = pivot.Right()
	//pivot.Right().parent = inner
	pivot.right = &inner.cordNode
	//inner.parent = pivot
	//pivot.parent = nil
	inner.adjustHeight()
	pivot.adjustHeight() // sequence matters
	//dump(&pivot.cordNode)
	return pivot
}

const balanceThres int = 3

func unbalanced(inner *innerNode) bool {
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
	//n.parent = inner.parent
	return n
}

func cloneLeaf(leaf *leafNode) *leafNode {
	l := makeLeafNode()
	l.leaf = leaf.leaf
	//l.parent = leaf.parent
	return l
}

func cloneNode(node *cordNode) *cordNode {
	if node == nil {
		return nil
	}
	if node.IsLeaf() {
		return &cloneLeaf(node.AsLeaf()).cordNode
	}
	return &clone(node.AsNode()).cordNode
}

// --- Helpers ---------------------------------------------------------------

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
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
