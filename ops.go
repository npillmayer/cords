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
	c1root.attachRight(c2.root.left) // c2 will remain unchanged, except parent
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
	// TODO case i == 0 or i == len(cord)
	leaf, inx, err := c.index(i)
	if err != nil {
		return c, Cord{}, err
	}
	parent := leaf.parent
	//var sibling *leafNode
	//if inx < leaf.Weight()-1 { // split mid-string
	if inx > 0 { // split mid-string
		p := split(leaf, inx)        // p is parent of new leaf and its left sibling
		p.parent = parent            // hook it up to orig leaf's parent
		parent = p                   // now call p the new leaf's parent
		leaf = parent.right.AsLeaf() // and call new leaf the leaf
		//sibling = parent.left.AsLeaf() // must have a left sibling
	}
	// else if lft := parent.Left(); lft != nil {
	// 	if sibling = lft.AsLeaf(); sibling == leaf {
	// 		sibling = nil
	// 	}
	// }
	// now leaf and parent are set ⇒ split it before leaf
	root1 := unzip(leaf) // produce a copy of all upward nodes until new left root
	var root2 *cordNode
	// leaf may be left or right child of parent
	n := &leaf.cordNode
	// search upwards until n is right child of its parent => split point
	for n != &root1.cordNode { // walk upwards upto root
		if n.parent.right == n { // yes, n is right child
			break // stop walking
		}
		n = &n.parent.cordNode
	}
	p := n.parent
	p.right = nil  // split off n
	n.parent = nil // split off n
	root2 = n      // build up root2, starting with n
	weight := n.Weight()
	for p != nil { // walk upwards again, searching for right children
		p.weight -= weight
		if p.right != nil {
			weight += length(p.right)
			rt := clone(p.right.AsNode())
			rt.parent = nil
			p.right = nil
			root2 = concat(root2, &rt.cordNode)
		}
		p = p.parent
	}
	// TODO correctly pack roots into Cords
	return Cord{root: root1}, Cord{root: root2.AsNode()}, nil
}

// ---------------------------------------------------------------------------

func concat(n1, n2 *cordNode) *cordNode {
	if n1 == nil {
		return n2
	} else if n2 == nil {
		return n1
	}
	inner := makeInnerNode()
	inner.attachLeft(n1)
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

func unzip(leaf *leafNode) *innerNode {
	l := cloneLeaf(leaf)
	inner := l.parent
	for inner != nil {
		inner = clone(inner)
		inner.adjustHeight()
		inner = inner.parent
	}
	return inner
}

func split(leaf *leafNode, i uint64) *innerNode {
	str := leaf.String()
	lstr := str[:i]
	rstr := str[i:]
	inner := makeInnerNode()
	inner.attachLeft(&makeStringLeaf(lstr).cordNode)
	inner.attachRight(&makeStringLeaf(rstr).cordNode)
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
	pivot.Left().parent = inner
	pivot.left = &inner.cordNode
	inner.parent = pivot
	pivot.parent = nil
	inner.adjustHeight()
	pivot.adjustHeight() // sequence matters
	return pivot
}

func rotateRight(inner *innerNode) *innerNode {
	T().Debugf("rotate right")
	pivot := clone(inner.left.AsNode()) // clone pivot and inner; copy on write
	inner = clone(inner)
	inner.left = pivot.Right()
	pivot.Right().parent = inner
	pivot.right = &inner.cordNode
	inner.parent = pivot
	pivot.parent = nil
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
	n.parent = inner.parent
	return n
}

func cloneLeaf(leaf *leafNode) *leafNode {
	l := makeLeafNode()
	l.leaf = leaf.leaf
	l.parent = leaf.parent
	return l
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
