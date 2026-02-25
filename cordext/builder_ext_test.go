package cords

import (
	"errors"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestNewBuilderWithExtensionRejectsNil(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	_, err := NewBuilderWithExtension[uint64](nil)
	if !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments, got %v", err)
	}
}

func TestBuilderWithExtensionBuildsCord(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	b, err := NewBuilderWithExtension[uint64](newlineExt{})
	if err != nil {
		t.Fatalf("NewBuilderWithExtension failed: %v", err)
	}
	_ = b.AppendString("Hello\n")
	_ = b.AppendString("World\n")
	cord := b.Cord()
	if cord.String() != "Hello\nWorld\n" {
		t.Fatalf("unexpected string: %q", cord.String())
	}
	ext, ok := cord.Ext()
	if !ok {
		t.Fatalf("Ext() not available")
	}
	if ext != 2 {
		t.Fatalf("unexpected extension value: got=%d want=2", ext)
	}
}

func TestBuilderWithExtensionDisallowsMutationAfterCord(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	b, err := NewBuilderWithExtension[uint64](newlineExt{})
	if err != nil {
		t.Fatalf("NewBuilderWithExtension failed: %v", err)
	}
	_ = b.AppendString("x")
	_ = b.Cord()
	if err := b.AppendString("y"); !errors.Is(err, ErrCordCompleted) {
		t.Fatalf("expected ErrCordCompleted, got %v", err)
	}
}

func TestBuilderWithExtensionResetAllowsReuse(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	b, err := NewBuilderWithExtension[uint64](newlineExt{})
	if err != nil {
		t.Fatalf("NewBuilderWithExtension failed: %v", err)
	}
	_ = b.AppendString("a\nb\n")
	first := b.Cord()
	ext, _ := first.Ext()
	if ext != 2 {
		t.Fatalf("unexpected first extension value: got=%d want=2", ext)
	}

	b.Reset()
	_ = b.AppendString("z\n")
	second := b.Cord()
	if second.String() != "z\n" {
		t.Fatalf("unexpected second string: %q", second.String())
	}
	ext, _ = second.Ext()
	if ext != 1 {
		t.Fatalf("unexpected second extension value: got=%d want=1", ext)
	}
}
