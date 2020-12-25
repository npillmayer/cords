package cords

import (
	"bytes"
	"errors"
)

// Cord is a type for an enhanced string.
type Cord struct {
	root *cordNode
}

// FromString creates a cord from a Go string.
func FromString(s string) Cord {
	r := &cordNode{
		weight:  uint64(len(s)),
		balance: -1,
	}
	leaf := &stringLeaf{
		index:  uint64(len(s)),
		parent: r,
		str:    s,
	}
	r.left = leaf
	return Cord{root: r}
}

func (cord Cord) String() string {
	var bf bytes.Buffer
	var err error
	err = cord.Each(func(node CordNode) error {
		if node.Left() == nil && node.Right() == nil {
			if _, err = bf.WriteString(node.String()); err != nil {
				T().Errorf(err.Error())
				return err
			}
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

// Each iterates over all nodes of the cord.
func (cord Cord) Each(f func(node CordNode) error) error {
	err := traverse(cord.root, f)
	return err
}

// EachLeaf iterates over all leaf nodes of the cord.
func (cord Cord) EachLeaf(f func(node CordNode) error) error {
	var err error
	err = traverse(cord.root, func(node CordNode) (e error) {
		if node.Left() == nil && node.Right() == nil {
			e = f(node)
		}
		return
	})
	return err
}

// traverse walks a cord in in-order.
func traverse(node CordNode, f func(CordNode) error) error {
	if l := node.Left(); l != nil {
		if err := traverse(l, f); err != nil {
			return err
		}
	}
	if err := f(node); err != nil {
		return err
	}
	if r := node.Right(); r != nil {
		if err := traverse(r, f); err != nil {
			return err
		}
	}
	return nil
}

var errIndexOutOfBounds error = errors.New("index out of bounds")

func (cord Cord) index(i uint64) (CordNode, uint64, error) {
	if cord.root == nil {
		return nil, 0, errIndexOutOfBounds
	}
	return index(cord.root, i)
}

// index finds the leaf node of a cord which contains a given index.
// Return values are the leaf node, the index within the leaf, and a possible error.
func index(node CordNode, i uint64) (CordNode, uint64, error) {
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

// CordNode is an interface type for nodes of a cord structure / binary tree.
type CordNode interface {
	Left() CordNode
	Right() CordNode
	Parent() CordNode
	Weight() uint64
	String() string
}

// ---------------------------------------------------------------------------

type cordNode struct {
	left, right CordNode
	parent      CordNode
	weight      uint64
	balance     int
}

func (node *cordNode) Weight() uint64 {
	return node.weight
}

func (node *cordNode) Left() CordNode {
	return node.left
}

func (node *cordNode) Right() CordNode {
	return node.right
}

func (node *cordNode) Parent() CordNode {
	return node.parent
}

func (node *cordNode) String() string {
	return ""
}

var _ CordNode = &cordNode{}

// ---------------------------------------------------------------------------

type stringLeaf struct {
	index  uint64
	parent CordNode
	str    string
}

func (leaf stringLeaf) Weight() uint64 {
	return leaf.index
}

func (leaf stringLeaf) Left() CordNode {
	return nil
}

func (leaf stringLeaf) Right() CordNode {
	return nil
}

func (leaf stringLeaf) Parent() CordNode {
	return leaf.parent
}

func (leaf stringLeaf) String() string {
	return leaf.str
}

var _ CordNode = &stringLeaf{}
