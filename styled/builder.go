package styled

import (
	"github.com/npillmayer/cords"
)

// TextBuilder is for building styled text from style runs.
type TextBuilder struct {
	cordBuilder cords.Builder
	text        *Text
	length      uint64
	done        bool
	styles      []styleSpan
}

type styleSpan struct {
	style Style
	span  span
}

// NewTextBuilder creates a new and empty builder for styled.Text.
func NewTextBuilder() *TextBuilder {
	return &TextBuilder{}
}

// Text returns the styled text which this builder is holding up to now.
// It is illegal to continue adding fragments after `Text` has been called,
// but `Text` may be called multiple times.
//
func (b TextBuilder) Text() *Text {
	b.done = true
	b.text = TextFromCord(b.cordBuilder.Cord())
	if b.text.Raw().IsVoid() {
		T().Debugf("cord builder: cord is void")
		return b.text
	}
	for _, s := range b.styles {
		b.text.Style(s.style, s.span.l, s.span.r)
	}
	return b.text
}

// Append appends a text fragement represented by a cord leaf at the end
// of the cord to build.
func (b *TextBuilder) Append(leaf cords.Leaf, style Style) error {
	if b.done {
		return cords.ErrCordCompleted
	}
	if leaf == nil || leaf.Weight() == 0 {
		return nil
	}
	b.cordBuilder.Append(leaf)
	b.styles = append(b.styles, styleSpan{style: style, span: toSpan(b.length, leaf.Weight())})
	b.length += leaf.Weight()
	return nil
}
