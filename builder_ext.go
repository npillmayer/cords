package cords

import (
	"unicode/utf8"

	"github.com/npillmayer/cords/chunk"
)

// BuilderEx incrementally stages text and finalizes it into a CordEx.
//
// BuilderEx behaves like Builder but materializes extension-enabled cord snapshots.
type BuilderEx[E any] struct {
	// front keeps prepended chunks in reverse logical order.
	front []chunk.Chunk
	// back keeps appended chunks in logical order.
	back []chunk.Chunk

	ext TextSegmentExtension[E]

	done  bool
	dirty bool
	cord  CordEx[E]
}

// NewBuilderWithExtension creates a new and empty extension-enabled cord builder.
func NewBuilderWithExtension[E any](ext TextSegmentExtension[E]) (*BuilderEx[E], error) {
	if ext == nil {
		return nil, ErrIllegalArguments
	}
	return &BuilderEx[E]{ext: ext}, nil
}

// Cord returns the cord built from all staged fragments.
//
// It is illegal to continue adding fragments after Cord has been called, but
// Cord may be called multiple times. Repeated calls return the same value until
// Reset is called.
func (b *BuilderEx[E]) Cord() CordEx[E] {
	if b == nil {
		return CordEx[E]{}
	}
	if b.dirty {
		b.cord = b.buildCord()
		b.dirty = false
	}
	b.done = true
	if b.cord.IsVoid() {
		tracer().Debugf("cord extension builder: cord is void")
	}
	return b.cord
}

// Reset drops the staged build and prepares the builder for a fresh build.
func (b *BuilderEx[E]) Reset() {
	b.front = nil
	b.back = nil
	b.done = false
	b.dirty = false
	b.cord = CordEx[E]{ext: b.ext}
}

// AppendString appends UTF-8 text to the staged build.
//
// Returns ErrCordCompleted if Cord() has already been called.
func (b *BuilderEx[E]) AppendString(text string) error {
	if !utf8.ValidString(text) {
		return chunk.ErrInvalidUTF8
	}
	return b.AppendBytes([]byte(text))
}

// PrependString prepends UTF-8 text to the staged build.
//
// Returns ErrCordCompleted if Cord() has already been called.
func (b *BuilderEx[E]) PrependString(text string) error {
	if !utf8.ValidString(text) {
		return chunk.ErrInvalidUTF8
	}
	return b.PrependBytes([]byte(text))
}

// AppendBytes appends UTF-8 bytes to the staged build.
//
// Adjacent chunks may be coalesced when capacity permits.
func (b *BuilderEx[E]) AppendBytes(text []byte) error {
	if b == nil || b.ext == nil {
		return ErrIllegalArguments
	}
	if b.done {
		return ErrCordCompleted
	}
	chunks, err := splitToChunks(text)
	if err != nil {
		return err
	}
	for _, c := range chunks {
		if len(b.back) > 0 {
			last := len(b.back) - 1
			merged, ok := b.back[last].Append(c.AsSlice())
			if ok {
				b.back[last] = merged
				continue
			}
		}
		b.back = append(b.back, c)
	}
	if len(chunks) > 0 {
		b.dirty = true
	}
	return nil
}

// PrependBytes prepends UTF-8 bytes to the staged build.
func (b *BuilderEx[E]) PrependBytes(text []byte) error {
	if b == nil || b.ext == nil {
		return ErrIllegalArguments
	}
	if b.done {
		return ErrCordCompleted
	}
	chunks, err := splitToChunks(text)
	if err != nil {
		return err
	}
	// front is stored in reverse logical order.
	for i := len(chunks) - 1; i >= 0; i-- {
		b.front = append(b.front, chunks[i])
	}
	if len(chunks) > 0 {
		b.dirty = true
	}
	return nil
}

// AppendChunk appends a pre-built chunk.
//
// Adjacent chunks may be coalesced when capacity permits.
func (b *BuilderEx[E]) AppendChunk(c chunk.Chunk) error {
	if b == nil || b.ext == nil {
		return ErrIllegalArguments
	}
	if b.done {
		return ErrCordCompleted
	}
	if c.IsEmpty() {
		return nil
	}
	if len(b.back) > 0 {
		last := len(b.back) - 1
		merged, ok := b.back[last].Append(c.AsSlice())
		if ok {
			b.back[last] = merged
			b.dirty = true
			return nil
		}
	}
	b.back = append(b.back, c)
	b.dirty = true
	return nil
}

// PrependChunk prepends a pre-built chunk.
func (b *BuilderEx[E]) PrependChunk(c chunk.Chunk) error {
	if b == nil || b.ext == nil {
		return ErrIllegalArguments
	}
	if b.done {
		return ErrCordCompleted
	}
	if c.IsEmpty() {
		return nil
	}
	b.front = append(b.front, c)
	b.dirty = true
	return nil
}

// buildCord materializes the current staged chunk sequence into a tree-backed CordEx.
func (b *BuilderEx[E]) buildCord() CordEx[E] {
	parts := b.orderedChunks()
	if len(parts) == 0 {
		return CordEx[E]{ext: b.ext}
	}
	tree, err := newChunkTreeEx(b.ext)
	assert(err == nil, "extension builder: btree.New failed")
	tree, err = tree.InsertAt(0, parts...)
	assert(err == nil, "extension builder: btree.InsertAt failed")
	return cordExFromTree(tree, b.ext)
}

// orderedChunks returns staged chunks in final logical order: prepends then appends.
func (b *BuilderEx[E]) orderedChunks() []chunk.Chunk {
	total := len(b.front) + len(b.back)
	if total == 0 {
		return nil
	}
	out := make([]chunk.Chunk, 0, total)
	for i := len(b.front) - 1; i >= 0; i-- {
		out = append(out, b.front[i])
	}
	out = append(out, b.back...)
	return out
}
