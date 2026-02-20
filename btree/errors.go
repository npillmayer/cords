package btree

import "errors"

var (
	// ErrInvalidConfig signals an invalid tree configuration.
	ErrInvalidConfig = errors.New("btree: invalid configuration")
	// ErrIndexOutOfBounds signals an invalid positional index.
	ErrIndexOutOfBounds = errors.New("btree: index out of bounds")
	// ErrUnimplemented marks API stubs that are intentionally not implemented yet.
	ErrUnimplemented = errors.New("btree: operation not implemented")
	// ErrInvalidDimension signals an invalid or missing dimension configuration.
	ErrInvalidDimension = errors.New("btree: invalid dimension")
	// ErrIncompatibleExtension signals that two trees have incompatible extension configs.
	ErrIncompatibleExtension = errors.New("btree: incompatible extension")
	// ErrExtensionUnavailable signals that an extension-specific API was used
	// without an extension configured for the tree.
	ErrExtensionUnavailable = errors.New("btree: extension unavailable")
)
