package cordext

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestBridgeFromTreeAndTree(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cordext")
	defer teardown()

	c := FromStringNoExt("abc🙂")
	tree := c.Tree()
	if tree == nil {
		t.Fatalf("expected non-nil tree")
	}
	wrapped := FromTreeNoExt(tree)
	if wrapped.String() != c.String() {
		t.Fatalf("wrapped string mismatch: got=%q want=%q", wrapped.String(), c.String())
	}
	if wrapped.Tree() != tree {
		t.Fatalf("expected wrapped tree pointer to be shared")
	}
}

func TestBridgeFromTreeEmpty(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cordext")
	defer teardown()

	wrapped := FromTreeNoExt(nil)
	if !wrapped.IsVoid() {
		t.Fatalf("expected void cord for nil tree")
	}
}
