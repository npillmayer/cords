package cordext

import (
	"bytes"
	"fmt"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
)

// TextSegmentExtension defines extension aggregation for cords over TextSegment.
//
// Implementations should be deterministic and side-effect free.
type TextSegmentExtension[E any] interface {
	// MagicID returns a stable identifier for extension semantics.
	MagicID() string
	// Zero returns the neutral element of extension aggregation.
	Zero() E
	// FromSegment projects one text segment into extension space.
	FromSegment(TextSegment) E
	// Add combines two extension summaries.
	Add(E, E) E
}

type textSegmentExtensionAdapter[E any] struct {
	ext TextSegmentExtension[E]
}

func (a textSegmentExtensionAdapter[E]) MagicID() string { return a.ext.MagicID() }
func (a textSegmentExtensionAdapter[E]) Zero() E         { return a.ext.Zero() }
func (a textSegmentExtensionAdapter[E]) Add(left, right E) E {
	return a.ext.Add(left, right)
}
func (a textSegmentExtensionAdapter[E]) FromItem(c chunk.Chunk, s chunk.Summary) E {
	return a.ext.FromSegment(newTextSegmentWithSummary(c, s))
}

// CordEx stores immutable UTF-8 text fragments in a persistent summarized tree
// with an extension summary E.
//
// Use FromStringWithExtension to construct configured values.
// A zero-value CordEx has no extension config and is only useful as an empty value.
type CordEx[E any] struct {
	tree *btree.Tree[chunk.Chunk, chunk.Summary, E]
	ext  TextSegmentExtension[E]
}

// FromStringNoExt creates a no-extension cord from a Go string.
func FromStringNoExt(s string) CordEx[btree.NO_EXT] {
	parts, err := splitToChunks([]byte(s))
	assert(err == nil, "cordext.FromStringNoExt requires valid UTF-8 input")
	cfg := btree.Config[chunk.Chunk, chunk.Summary, btree.NO_EXT]{Monoid: chunk.Monoid{}}
	tree, err := btree.New[chunk.Chunk, chunk.Summary](cfg)
	assert(err == nil, "cordext.FromStringNoExt: cannot create chunk tree")
	if len(parts) > 0 {
		tree, err = tree.InsertAt(0, parts...)
		assert(err == nil, "cordext.FromStringNoExt: cannot insert chunks")
	}
	return cordExFromTree(tree, nil)
}

// FromTreeNoExt wraps a no-extension chunk tree as a no-extension CordEx.
//
// The tree is shared (persistent data-structure semantics apply).
func FromTreeNoExt(tree *btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT]) CordEx[btree.NO_EXT] {
	if tree == nil || tree.IsEmpty() {
		return CordEx[btree.NO_EXT]{}
	}
	return CordEx[btree.NO_EXT]{tree: tree}
}

// FromStringWithExtension creates a cord with extension aggregation.
func FromStringWithExtension[E any](s string, ext TextSegmentExtension[E]) (CordEx[E], error) {
	parts, err := splitToChunks([]byte(s))
	if err != nil {
		return CordEx[E]{}, err
	}
	tree, err := newChunkTreeEx(ext)
	if err != nil {
		return CordEx[E]{}, err
	}
	if len(parts) > 0 {
		tree, err = tree.InsertAt(0, parts...)
		if err != nil {
			return CordEx[E]{}, err
		}
	}
	return cordExFromTree(tree, ext), nil
}

func newChunkTreeEx[E any](ext TextSegmentExtension[E]) (*btree.Tree[chunk.Chunk, chunk.Summary, E], error) {
	if ext == nil {
		return nil, ErrIllegalArguments
	}
	cfg := btree.Config[chunk.Chunk, chunk.Summary, E]{
		Monoid:    chunk.Monoid{},
		Extension: textSegmentExtensionAdapter[E]{ext: ext},
	}
	return btree.New[chunk.Chunk, chunk.Summary](cfg)
}

func treeFromCordEx[E any](cord CordEx[E]) (*btree.Tree[chunk.Chunk, chunk.Summary, E], error) {
	if cord.tree != nil {
		return cord.tree, nil
	}
	cfg := btree.Config[chunk.Chunk, chunk.Summary, E]{
		Monoid: chunk.Monoid{},
	}
	if cord.ext != nil {
		cfg.Extension = textSegmentExtensionAdapter[E]{ext: cord.ext}
	}
	return btree.New[chunk.Chunk, chunk.Summary](cfg)
}

func cordExFromTree[E any](tree *btree.Tree[chunk.Chunk, chunk.Summary, E], ext TextSegmentExtension[E]) CordEx[E] {
	return CordEx[E]{tree: tree, ext: ext}
}

func newEmptyLike[E any](tree *btree.Tree[chunk.Chunk, chunk.Summary, E]) (*btree.Tree[chunk.Chunk, chunk.Summary, E], error) {
	cfg := tree.Config()
	return btree.New[chunk.Chunk, chunk.Summary](cfg)
}

// Extension returns the configured extension implementation.
func (cord CordEx[E]) Extension() TextSegmentExtension[E] {
	return cord.ext
}

// Tree returns the underlying chunk tree.
//
// This is intended for package-bridge scenarios.
func (cord CordEx[E]) Tree() *btree.Tree[chunk.Chunk, chunk.Summary, E] {
	return cord.tree
}

// String returns the complete cord as a Go string.
func (cord CordEx[E]) String() string {
	if cord.tree == nil || cord.tree.IsEmpty() {
		return ""
	}
	var bf bytes.Buffer
	cord.tree.ForEachItem(func(c chunk.Chunk) bool {
		_, _ = bf.WriteString(c.String())
		return true
	})
	return bf.String()
}

// IsVoid reports whether the cord has no bytes.
func (cord CordEx[E]) IsVoid() bool {
	return cord.tree == nil || cord.tree.IsEmpty()
}

// Len returns the cord length in bytes.
func (cord CordEx[E]) Len() uint64 {
	if cord.tree == nil {
		return 0
	}
	return cord.tree.Summary().Bytes
}

// Summary returns bytes/chars/lines summary for this cord.
func (cord CordEx[E]) Summary() chunk.Summary {
	if cord.tree == nil {
		return chunk.Summary{}
	}
	return cord.tree.Summary()
}

// Ext returns the total extension summary for this cord.
func (cord CordEx[E]) Ext() (E, bool) {
	if cord.tree == nil {
		var zero E
		return zero, false
	}
	return cord.tree.Ext()
}

// PrefixExt returns the aggregated extension value for chunk range [0,itemIndex).
func (cord CordEx[E]) PrefixExt(itemIndex int64) (E, error) {
	tree, err := treeFromCordEx(cord)
	if err != nil {
		var zero E
		return zero, err
	}
	return tree.PrefixExt(itemIndex)
}

// Concat concatenates cords and returns a new extension-enabled cord.
func (cord CordEx[E]) Concat(others ...CordEx[E]) (CordEx[E], error) {
	return concatTreeEx(cord, others...)
}

// Insert inserts c into cord at byte offset i.
func (cord CordEx[E]) Insert(c CordEx[E], i uint64) (CordEx[E], error) {
	return insertTreeEx(cord, c, i)
}

// Split splits cord right before byte offset i.
func (cord CordEx[E]) Split(i uint64) (CordEx[E], CordEx[E], error) {
	return splitTreeEx(cord, i)
}

// Cut removes byte range [i, i+l) and returns (remaining, removed).
func (cord CordEx[E]) Cut(i, l uint64) (CordEx[E], CordEx[E], error) {
	return cutTreeEx(cord, i, l)
}

// Report materializes l bytes at offset i as a Go string.
func (cord CordEx[E]) Report(i, l uint64) (string, error) {
	return reportTreeEx(cord, i, l)
}

// Substr returns a new cord for byte range [i, i+l).
func (cord CordEx[E]) Substr(i, l uint64) (CordEx[E], error) {
	return substrTreeEx(cord, i, l)
}

// Index returns the chunk containing byte i and the local chunk offset.
func (cord CordEx[E]) Index(i uint64) (chunk.Chunk, uint64, error) {
	return indexTreeEx(cord, i)
}

// FragmentCount returns the number of chunks currently stored in the cord.
func (cord CordEx[E]) FragmentCount() int {
	cnt := 0
	_ = cord.EachChunk(func(chunk.Chunk, uint64) error {
		cnt++
		return nil
	})
	return cnt
}

// NewExtCursor creates an extension cursor over a CordEx.
func NewExtCursor[E any, K any](cord CordEx[E], dim btree.Dimension[E, K]) (*btree.ExtCursor[chunk.Chunk, chunk.Summary, E, K], error) {
	tree, err := treeFromCordEx(cord)
	if err != nil {
		return nil, err
	}
	return btree.NewExtCursor[chunk.Chunk, chunk.Summary, E, K](tree, dim)
}

func concatTreeEx[E any](cord CordEx[E], others ...CordEx[E]) (CordEx[E], error) {
	base, err := treeFromCordEx(cord)
	if err != nil {
		return CordEx[E]{}, err
	}
	ext := cord.ext
	for _, c := range others {
		other, convErr := treeFromCordEx(c)
		if convErr != nil {
			return CordEx[E]{}, convErr
		}
		base, err = base.Concat(other)
		if err != nil {
			return CordEx[E]{}, err
		}
		if ext == nil && c.ext != nil {
			ext = c.ext
		}
	}
	return cordExFromTree(base, ext), nil
}

func splitTreeEx[E any](cord CordEx[E], i uint64) (CordEx[E], CordEx[E], error) {
	tree, err := treeFromCordEx(cord)
	if err != nil {
		return CordEx[E]{}, CordEx[E]{}, err
	}
	left, right, err := splitTreeByByteEx(tree, i)
	if err != nil {
		return CordEx[E]{}, CordEx[E]{}, err
	}
	return cordExFromTree(left, cord.ext), cordExFromTree(right, cord.ext), nil
}

func splitTreeByByteEx[E any](tree *btree.Tree[chunk.Chunk, chunk.Summary, E], i uint64) (*btree.Tree[chunk.Chunk, chunk.Summary, E], *btree.Tree[chunk.Chunk, chunk.Summary, E], error) {
	total := tree.Summary().Bytes
	if i > total {
		return nil, nil, ErrIndexOutOfBounds
	}
	if i == 0 {
		empty, err := newEmptyLike(tree)
		if err != nil {
			return nil, nil, err
		}
		return empty, tree, nil
	}
	if i == total {
		empty, err := newEmptyLike(tree)
		if err != nil {
			return nil, nil, err
		}
		return tree, empty, nil
	}
	cursor, err := btree.NewCursor[chunk.Chunk, chunk.Summary, E, uint64](tree, chunk.ByteDimension{})
	if err != nil {
		return nil, nil, err
	}
	itemIndex, acc, err := cursor.Seek(i)
	if err != nil {
		return nil, nil, err
	}
	if itemIndex < 0 || itemIndex >= tree.Len() {
		return nil, nil, ErrIndexOutOfBounds
	}
	item, err := tree.At(itemIndex)
	if err != nil {
		return nil, nil, err
	}
	itemBytes := item.Summary().Bytes
	before := acc - itemBytes
	local := i - before
	if local == 0 {
		l, r, err := tree.SplitAt(itemIndex)
		return l, r, err
	}
	if local == itemBytes {
		l, r, err := tree.SplitAt(itemIndex + 1)
		return l, r, err
	}
	leftSlice, rightSlice, err := item.SplitAt(int(local))
	if err != nil {
		return nil, nil, fmt.Errorf("split index %d is not on UTF-8 boundary: %w", i, err)
	}
	leftChunk, err := chunk.NewBytes(leftSlice.Bytes())
	if err != nil {
		return nil, nil, err
	}
	rightChunk, err := chunk.NewBytes(rightSlice.Bytes())
	if err != nil {
		return nil, nil, err
	}
	left, right, err := tree.SplitAt(itemIndex)
	if err != nil {
		return nil, nil, err
	}
	right, err = right.DeleteAt(0)
	if err != nil {
		return nil, nil, err
	}
	left, err = left.InsertAt(left.Len(), leftChunk)
	if err != nil {
		return nil, nil, err
	}
	right, err = right.InsertAt(0, rightChunk)
	if err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

func insertTreeEx[E any](cord CordEx[E], c CordEx[E], i uint64) (CordEx[E], error) {
	if cord.IsVoid() && i == 0 {
		return c, nil
	}
	if cord.Len() < i {
		return CordEx[E]{}, ErrIndexOutOfBounds
	}
	if c.IsVoid() {
		return cord, nil
	}
	left, right, err := splitTreeEx(cord, i)
	if err != nil {
		return CordEx[E]{}, err
	}
	return concatTreeEx(left, c, right)
}

func cutTreeEx[E any](cord CordEx[E], i, l uint64) (CordEx[E], CordEx[E], error) {
	if l == 0 {
		return cord, CordEx[E]{ext: cord.ext}, nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return CordEx[E]{}, CordEx[E]{}, ErrIndexOutOfBounds
	}
	left, rest, err := splitTreeEx(cord, i)
	if err != nil {
		return CordEx[E]{}, CordEx[E]{}, err
	}
	mid, right, err := splitTreeEx(rest, l)
	if err != nil {
		return CordEx[E]{}, CordEx[E]{}, err
	}
	out, err := concatTreeEx(left, right)
	if err != nil {
		return CordEx[E]{}, CordEx[E]{}, err
	}
	return out, mid, nil
}

func substrTreeEx[E any](cord CordEx[E], i, l uint64) (CordEx[E], error) {
	if l == 0 {
		return CordEx[E]{ext: cord.ext}, nil
	}
	if cord.Len() < i || cord.Len() < i+l {
		return CordEx[E]{}, ErrIndexOutOfBounds
	}
	_, rest, err := splitTreeEx(cord, i)
	if err != nil {
		return CordEx[E]{}, err
	}
	sub, _, err := splitTreeEx(rest, l)
	if err != nil {
		return CordEx[E]{}, err
	}
	return sub, nil
}

func reportTreeEx[E any](cord CordEx[E], i, l uint64) (string, error) {
	sub, err := substrTreeEx(cord, i, l)
	if err != nil {
		return "", err
	}
	return sub.String(), nil
}

func indexTreeEx[E any](cord CordEx[E], i uint64) (chunk.Chunk, uint64, error) {
	tree, err := treeFromCordEx(cord)
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	if i >= tree.Summary().Bytes {
		return chunk.Chunk{}, 0, ErrIndexOutOfBounds
	}
	cursor, err := btree.NewCursor[chunk.Chunk, chunk.Summary, E, uint64](tree, chunk.ByteDimension{})
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	itemIndex, acc, err := cursor.Seek(i + 1)
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	item, err := tree.At(itemIndex)
	if err != nil {
		return chunk.Chunk{}, 0, err
	}
	before := acc - item.Summary().Bytes
	return item, i - before, nil
}
