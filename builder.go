package cords

import (
	"unicode/utf8"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
)

// Builder incrementally stages text and finalizes it into a Cord.
//
// Builder collects UTF-8 text as fixed-size chunks and materializes the cord
// only when Cord() is called. This keeps mutation logic in one place and makes
// migration to the new btree backend straightforward.
//
// The empty instance is a valid builder, but clients may use NewBuilder.
type Builder struct {
	// front keeps prepended chunks in reverse logical order.
	front []chunk.Chunk
	// back keeps appended chunks in logical order.
	back []chunk.Chunk

	done  bool
	dirty bool
	cord  Cord
}

// NewBuilder creates a new and empty cord builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Cord returns the cord built from all staged fragments.
//
// It is illegal to continue adding fragments after Cord has been called, but
// Cord may be called multiple times.
func (b *Builder) Cord() Cord {
	if b == nil {
		return Cord{}
	}
	if b.dirty {
		b.cord = b.buildCord()
		b.dirty = false
	}
	b.done = true
	if b.cord.IsVoid() {
		tracer().Debugf("cord builder: cord is void")
	}
	return b.cord
}

// Reset drops the staged build and prepares the builder for a fresh build.
func (b *Builder) Reset() {
	b.front = nil
	b.back = nil
	b.done = false
	b.dirty = false
	b.cord = Cord{}
}

// AppendString appends UTF-8 text to the staged build.
func (b *Builder) AppendString(text string) error {
	if !utf8.ValidString(text) {
		return chunk.ErrInvalidUTF8
	}
	return b.AppendBytes([]byte(text))
}

// PrependString prepends UTF-8 text to the staged build.
func (b *Builder) PrependString(text string) error {
	if !utf8.ValidString(text) {
		return chunk.ErrInvalidUTF8
	}
	return b.PrependBytes([]byte(text))
}

// AppendBytes appends UTF-8 bytes to the staged build.
func (b *Builder) AppendBytes(text []byte) error {
	if b == nil {
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
func (b *Builder) PrependBytes(text []byte) error {
	if b == nil {
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
func (b *Builder) AppendChunk(c chunk.Chunk) error {
	if b == nil {
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
func (b *Builder) PrependChunk(c chunk.Chunk) error {
	if b == nil {
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

func (b *Builder) buildCord() Cord {
	parts := b.orderedChunks()
	if len(parts) == 0 {
		return Cord{}
	}
	cfg := btree.Config[chunk.Summary]{Monoid: chunk.Monoid{}}
	tree, err := btree.New[chunk.Chunk, chunk.Summary](cfg)
	assert(err == nil, "builder: btree.New failed")
	tree, err = tree.InsertAt(0, parts...)
	assert(err == nil, "builder: btree.InsertAt failed")
	return cordFromTree(tree)
}

func (b *Builder) orderedChunks() []chunk.Chunk {
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

// splitToChunks splits UTF-8 bytes into chunk-sized pieces.
//
// Boundaries are adjusted so no chunk starts or ends in the middle of a UTF-8
// rune. This mirrors chunk.NewBytes requirements for ingest pipelines.
func splitToChunks(text []byte) ([]chunk.Chunk, error) {
	if len(text) == 0 {
		return nil, nil
	}
	if !utf8.Valid(text) {
		return nil, chunk.ErrInvalidUTF8
	}
	parts := make([]chunk.Chunk, 0, 1+len(text)/chunk.MaxBase)
	for i := 0; i < len(text); {
		end := i + chunk.MaxBase
		if end >= len(text) {
			end = len(text)
		} else {
			for end > i && !utf8.RuneStart(text[end]) {
				end--
			}
			if end == i {
				return nil, chunk.ErrInvalidUTF8
			}
		}
		c, err := chunk.NewBytes(text[i:end])
		if err != nil {
			return nil, err
		}
		parts = append(parts, c)
		i = end
	}
	return parts, nil
}
