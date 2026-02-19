package chunk

import "errors"

var (
	// ErrInvalidUTF8 signals invalid UTF-8 source text.
	ErrInvalidUTF8 = errors.New("chunk: invalid UTF-8")
	// ErrChunkTooLarge signals that input exceeds MaxBase bytes.
	ErrChunkTooLarge = errors.New("chunk: text exceeds chunk capacity")
	// ErrIndexOutOfBounds signals invalid byte offsets for slicing/splitting.
	ErrIndexOutOfBounds = errors.New("chunk: index out of bounds")
	// ErrNotCharBoundary signals non-UTF-8-boundary offsets.
	ErrNotCharBoundary = errors.New("chunk: offset is not a char boundary")
)
