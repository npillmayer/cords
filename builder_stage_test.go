package cords

import (
	"errors"
	"strings"
	"testing"

	"github.com/npillmayer/cords/chunk"
)

func TestBuilderAppendAndPrependString(t *testing.T) {
	b := NewBuilder()
	if err := b.AppendString("name_is"); err != nil {
		t.Fatalf("AppendString failed: %v", err)
	}
	if err := b.PrependString("Hello_my_"); err != nil {
		t.Fatalf("PrependString failed: %v", err)
	}
	if err := b.AppendString("_Simon"); err != nil {
		t.Fatalf("AppendString failed: %v", err)
	}
	c := b.Cord()
	if c.tree == nil {
		t.Fatalf("expected builder result to carry btree storage")
	}
	if got, want := c.String(), "Hello_my_name_is_Simon"; got != want {
		t.Fatalf("unexpected cord string: got %q want %q", got, want)
	}
}

func TestBuilderSplitToChunksAtRuneBoundaries(t *testing.T) {
	prefix := strings.Repeat("a", chunk.MaxBase-1)
	input := prefix + "😀" + strings.Repeat("b", chunk.MaxBase+3)

	b := NewBuilder()
	if err := b.AppendString(input); err != nil {
		t.Fatalf("AppendString failed: %v", err)
	}
	c := b.Cord()
	if got := c.String(); got != input {
		t.Fatalf("builder changed input text across chunking; got %q want %q", got, input)
	}
}

func TestBuilderRejectsInvalidUTF8(t *testing.T) {
	b := NewBuilder()
	err := b.AppendBytes([]byte{0xff, 0xfe})
	if !errors.Is(err, chunk.ErrInvalidUTF8) {
		t.Fatalf("expected ErrInvalidUTF8, got %v", err)
	}
}

func TestBuilderDisallowsMutationAfterCord(t *testing.T) {
	b := NewBuilder()
	if err := b.AppendString("abc"); err != nil {
		t.Fatalf("AppendString failed: %v", err)
	}
	_ = b.Cord()
	if err := b.AppendString("def"); !errors.Is(err, ErrCordCompleted) {
		t.Fatalf("expected ErrCordCompleted, got %v", err)
	}
	if err := b.Prepend(StringLeaf("x")); !errors.Is(err, ErrCordCompleted) {
		t.Fatalf("expected ErrCordCompleted from Prepend, got %v", err)
	}
}

func TestBuilderResetAllowsReuse(t *testing.T) {
	b := NewBuilder()
	if err := b.AppendString("one"); err != nil {
		t.Fatalf("AppendString failed: %v", err)
	}
	_ = b.Cord()
	b.Reset()
	if err := b.AppendString("two"); err != nil {
		t.Fatalf("AppendString after Reset failed: %v", err)
	}
	c := b.Cord()
	if got := c.String(); got != "two" {
		t.Fatalf("unexpected cord after Reset: %q", got)
	}
}
