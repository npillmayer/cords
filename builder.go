package cords

// CordBuilder is for building cords by adding text fragments (leafs).
// The empty instance is a valid cord builder, but clients may use NewBuilder
// instead.
type CordBuilder struct {
	cord Cord
	done bool
}

// NewBuilder creates a new and empty cord builder.
func NewBuilder() *CordBuilder {
	return &CordBuilder{}
}

// Cord returns the cord which this builder is holding up to now.
// It is illegal to continue adding fragments after `Cord` has been called,
// but `Cord` may be called multiple times.
func (b CordBuilder) Cord() Cord {
	b.done = true
	if b.cord.IsVoid() {
		T().Debugf("cord builder: cord is void")
	}
	return b.cord
}

// Reset drops the cord building currently in progress an prepares the builder
// for a fresh build.
func (b *CordBuilder) Reset() {
	b.cord.root = nil
	b.done = false
}

// Append appends a text fragement represented by a cord leaf at the end
// of the cord to build.
func (b *CordBuilder) Append(leaf Leaf) error {
	if b.done {
		return ErrCordCompleted
	}
	if leaf == nil || leaf.Weight() == 0 {
		return nil
	}
	if b.cord.IsVoid() {
		b.cord = makeSingleLeafCord(b.cord, leaf)
		return nil
	}
	lnode := makeLeafNode()
	lnode.leaf = leaf
	if b.cord.root.right != nil {
		panic("inconsistency in cord-builder")
	}
	b.cord.root.attachRight(&lnode.cordNode)
	newroot := makeInnerNode()
	newroot.attachLeft(&b.cord.root.cordNode)
	b.cord.root = newroot
	return nil
}

// Prepend pushes a text fragement represented by a cord leaf at the beginning
// of the cord to build.
func (b *CordBuilder) Prepend(leaf Leaf) error {
	if b.done {
		return ErrCordCompleted
	}
	if leaf == nil || leaf.Weight() == 0 {
		return nil
	}
	if b.cord.IsVoid() {
		b.cord = makeSingleLeafCord(b.cord, leaf)
		return nil
	}
	lnode := makeLeafNode()
	lnode.leaf = leaf
	n := makeInnerNode()
	n.attachLeft(&lnode.cordNode)
	n.attachRight(b.cord.root.Left())
	b.cord.root.left, b.cord.root.right, b.cord.root.weight = nil, nil, 0
	b.cord.root.attachLeft(&n.cordNode)
	return nil
}

func (b CordBuilder) balance() {
	if b.cord.IsVoid() || b.cord.root.Left().IsLeaf() {
		panic("builder wants cord balanced, but cord has no inner nodes")
	}
	if unbalanced(b.cord.root.Left()) {
		bal := balance(b.cord.root.Left().AsInner())
		b.cord.root.attachLeft(&bal.cordNode)
	}
}

func makeSingleLeafCord(cord Cord, leaf Leaf) Cord {
	lnode := makeLeafNode()
	lnode.leaf = leaf
	cord.root = makeInnerNode()
	cord.root.attachLeft(&lnode.cordNode)
	return cord
}
