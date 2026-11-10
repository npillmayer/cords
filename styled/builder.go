package styled

import (
	"fmt"
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
)

// TextBuilder is for building styled text from style runs.
type TextBuilder struct {
	cordBuilder *cordext.BuilderEx[btree.NO_EXT]
	text        Text
	length      uint64
	done        bool
	styles      []styleSpan
}

type styleSpan struct {
	style Style
	span  span
}

func (b *TextBuilder) ensureBuilder() {
	if b.cordBuilder == nil {
		b.cordBuilder = cordext.NewBuilderNoExt()
	}
}

// NewTextBuilder creates a new and empty builder for styled.Text.
func NewTextBuilder() *TextBuilder {
	return &TextBuilder{
		cordBuilder: cordext.NewBuilderNoExt(),
	}
}

// Text returns the styled text which this builder is holding up to now.
// It is illegal to continue adding fragments after `Text` has been called,
// but `Text` may be called multiple times, returning the cached snapshot.
// A failure applying internally recorded spans panics rather than silently losing styles.
func (b *TextBuilder) Text() Text {
	if b.done {
		return b.text
	}
	b.ensureBuilder()
	b.done = true
	b.text = TextFromCord(b.cordBuilder.Cord())
	if b.text.Raw().IsVoid() {
		tracer().Debugf("cord builder: cord is void")
		return b.text
	}
	for _, s := range b.styles {
		var err error
		b.text, err = b.text.Style(s.style, s.span.l, s.span.r)
		if err != nil {
			panic(fmt.Errorf("styled.TextBuilder: applying recorded span [%d,%d): %w", s.span.l, s.span.r, err))
		}
	}
	return b.text
}

// Append appends a text fragement represented by a cord leaf at the end
// of the text to build.
func (b *TextBuilder) Append(chunk *chunk.Chunk, style Style) error {
	b.ensureBuilder()
	if b.done {
		return cordext.ErrCordCompleted
	}
	if chunk == nil || chunk.Len() == 0 {
		return nil
	}
	if err := b.cordBuilder.AppendChunk(*chunk); err != nil {
		return err
	}
	//T().Infof("Append leaf = %v (%d)", leaf, leaf.Weight())
	b.recordStyle(uint64(chunk.Len()), style)
	return nil
}

// Len returns the provisional total UTF-8 byte length of the appended fragments.
func (b *TextBuilder) Len() uint64 {
	return b.length
}

// AppendString appends UTF-8 text and its style. Empty strings are no-ops.
// Invalid UTF-8 is rejected without changing text, length, or styles.
func (b *TextBuilder) AppendString(s string, style Style) error {
	b.ensureBuilder()
	if b.done {
		return cordext.ErrCordCompleted
	}
	if err := b.cordBuilder.AppendString(s); err != nil {
		return err
	}
	b.recordStyle(uint64(len(s)), style)
	return nil
}

func (b *TextBuilder) recordStyle(length uint64, style Style) {
	if length == 0 {
		return
	}
	b.styles = append(b.styles, styleSpan{style: style, span: toSpan(b.length, b.length+length)})
	b.length += length
}
