package cords

import (
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
)

// Builder incrementally stages text and finalizes it into a Cord.
//
// Builder is now a thin wrapper over the lower-level no-extension builder in
// sub-package cordext.
type Builder struct {
	builder *cordext.BuilderEx[btree.NO_EXT]
}

func (b *Builder) ensure() *cordext.BuilderEx[btree.NO_EXT] {
	if b.builder == nil {
		b.builder = cordext.NewBuilderNoExt()
	}
	return b.builder
}

// NewBuilder creates a new and empty cord builder.
func NewBuilder() *Builder {
	return &Builder{builder: cordext.NewBuilderNoExt()}
}

// Cord returns the cord built from all staged fragments.
//
// It is illegal to continue adding fragments after Cord has been called, but
// Cord may be called multiple times. Repeated calls return the same value until
// Reset is called.
func (b *Builder) Cord() Cord {
	if b == nil {
		return Cord{}
	}
	return fromCordext(b.ensure().Cord())
}

// Reset drops the staged build and prepares the builder for a fresh build.
func (b *Builder) Reset() {
	if b == nil {
		return
	}
	b.ensure().Reset()
}

// AppendString appends UTF-8 text to the staged build.
//
// Returns ErrCordCompleted if Cord() has already been called.
func (b *Builder) AppendString(text string) error {
	if b == nil {
		return ErrIllegalArguments
	}
	return fromCordextError(b.ensure().AppendString(text))
}

// PrependString prepends UTF-8 text to the staged build.
//
// Returns ErrCordCompleted if Cord() has already been called.
func (b *Builder) PrependString(text string) error {
	if b == nil {
		return ErrIllegalArguments
	}
	return fromCordextError(b.ensure().PrependString(text))
}

// AppendBytes appends UTF-8 bytes to the staged build.
//
// Adjacent chunks may be coalesced when capacity permits.
func (b *Builder) AppendBytes(text []byte) error {
	if b == nil {
		return ErrIllegalArguments
	}
	return fromCordextError(b.ensure().AppendBytes(text))
}

// PrependBytes prepends UTF-8 bytes to the staged build.
func (b *Builder) PrependBytes(text []byte) error {
	if b == nil {
		return ErrIllegalArguments
	}
	return fromCordextError(b.ensure().PrependBytes(text))
}

// AppendChunk appends a pre-built chunk.
//
// Adjacent chunks may be coalesced when capacity permits.
func (b *Builder) AppendChunk(c chunk.Chunk) error {
	if b == nil {
		return ErrIllegalArguments
	}
	return fromCordextError(b.ensure().AppendChunk(c))
}

// PrependChunk prepends a pre-built chunk.
func (b *Builder) PrependChunk(c chunk.Chunk) error {
	if b == nil {
		return ErrIllegalArguments
	}
	return fromCordextError(b.ensure().PrependChunk(c))
}
