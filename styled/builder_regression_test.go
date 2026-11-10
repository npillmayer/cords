package styled_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
	"github.com/npillmayer/cords/styled"
)

type builderStyle string

func TestTextBuilderChunkStyles(t *testing.T) {
	b := styled.NewTextBuilder()
	c, err := chunk.New("é")
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Append(&c, builderStyle("bold")); err != nil {
		t.Fatal(err)
	}
	text := b.Text()
	got, _, err := text.StyleAt(0)
	if err != nil || got != builderStyle("bold") {
		t.Fatalf("chunk style lost: %v, %v", got, err)
	}
}

func (s builderStyle) String() string { return string(s) }
func (s builderStyle) Equals(other styled.Style) bool {
	v, ok := other.(builderStyle)
	return ok && v == s
}

func TestTextBuilderLengthAndStyles(t *testing.T) {
	for _, makeBuilder := range []func() *styled.TextBuilder{styled.NewTextBuilder, func() *styled.TextBuilder { return &styled.TextBuilder{} }} {
		b := makeBuilder()
		fragments := []struct {
			text    string
			style   styled.Style
			asChunk bool
		}{
			{"é", builderStyle("red"), false}, {"世", builderStyle("blue"), true},
			{" ", nil, false}, {strings.Repeat("abé", 40), builderStyle("red"), false},
			{"!", builderStyle("red"), true},
		}
		var raw string
		for _, f := range fragments {
			var err error
			if f.asChunk {
				c, e := chunk.New(f.text)
				if e != nil {
					t.Fatal(e)
				}
				err = b.Append(&c, f.style)
			} else {
				err = b.AppendString(f.text, f.style)
			}
			if err != nil {
				t.Fatal(err)
			}
			raw += f.text
			if b.Len() != uint64(len(raw)) {
				t.Errorf("Len=%d, want %d", b.Len(), len(raw))
			}
		}
		text := b.Text()
		if text.Raw().String() != raw || text.Raw().Len() != b.Len() {
			t.Fatal("text/length mismatch")
		}
		var pos uint64
		for _, f := range fragments {
			for i := range len(f.text) {
				got, _, err := text.StyleAt(pos + uint64(i))
				if err != nil || got != f.style {
					t.Fatalf("style at byte %d = %v (%v), want %v", pos+uint64(i), got, err, f.style)
				}
			}
			pos += uint64(len(f.text))
		}
		changed, err := text.Style(builderStyle("changed"), 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		_ = changed
		again := b.Text()
		got, _, err := again.StyleAt(0)
		if err != nil || got != builderStyle("red") {
			t.Fatal("cached snapshot lost original style")
		}
		c, _ := chunk.New("x")
		for _, err := range []error{b.AppendString("x", nil), b.AppendString("", nil), b.Append(&c, nil), b.Append(nil, nil)} {
			if !errors.Is(err, cordext.ErrCordCompleted) {
				t.Fatalf("append after Text: %v", err)
			}
		}
	}
}

func TestTextBuilderEmptyAndRejectedAppends(t *testing.T) {
	b := styled.NewTextBuilder()
	if err := b.AppendString("", builderStyle("empty")); err != nil {
		t.Fatal(err)
	}
	empty, _ := chunk.New("")
	if err := b.Append(&empty, nil); err != nil {
		t.Fatal(err)
	}
	if err := b.Append(nil, nil); err != nil {
		t.Fatal(err)
	}
	if b.Len() != 0 {
		t.Fatal("empty append changed length")
	}
	text := b.Text()
	again := b.Text()
	if !text.IsVoid() || !again.IsVoid() {
		t.Fatal("empty builder has content")
	}
	b = styled.NewTextBuilder()
	if err := b.AppendString("é", builderStyle("before")); err != nil {
		t.Fatal(err)
	}
	if err := b.AppendString("prefix\xff", builderStyle("invalid")); !errors.Is(err, chunk.ErrInvalidUTF8) {
		t.Fatalf("invalid UTF-8: %v", err)
	}
	if b.Len() != 2 {
		t.Fatalf("rejection changed length: %d", b.Len())
	}
	if err := b.AppendString("z", builderStyle("after")); err != nil {
		t.Fatal(err)
	}
	text = b.Text()
	if text.Raw().String() != "éz" {
		t.Fatal("rejected text leaked")
	}
	for pos, want := range map[uint64]builderStyle{0: "before", 2: "after"} {
		got, _, err := text.StyleAt(pos)
		if err != nil || got != want {
			t.Fatalf("style at %d: %v, %v", pos, got, err)
		}
	}
}
