package cords

import (
	"errors"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/cordext"
)

// toCordext converts a root Cord into a no-extension cordext value.
//
// This is an internal migration helper.
func toCordext(cord Cord) cordext.CordEx[btree.NO_EXT] {
	tree, err := treeFromCord(cord)
	assert(err == nil, "toCordext: cannot materialize tree")
	return cordext.FromTreeNoExt(tree)
}

// fromCordext converts a no-extension cordext value back into root Cord.
//
// This is an internal migration helper.
func fromCordext(cord cordext.CordEx[btree.NO_EXT]) Cord {
	return cordFromTree(cord.Tree())
}

// fromCordextError maps cordext package errors back to root package errors.
//
// This preserves root API error contracts while delegating implementation.
func fromCordextError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, cordext.ErrIndexOutOfBounds):
		return ErrIndexOutOfBounds
	case errors.Is(err, cordext.ErrIllegalArguments):
		return ErrIllegalArguments
	case errors.Is(err, cordext.ErrCordCompleted):
		return ErrCordCompleted
	default:
		return err
	}
}
