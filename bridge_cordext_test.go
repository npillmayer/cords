package cords

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestBridgeCordextRoundtrip(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	c := FromString("Hello🙂\nWorld")
	x := toCordext(c)
	if x.String() != c.String() {
		t.Fatalf("root->cordext bridge mismatch: root=%q ext=%q", c.String(), x.String())
	}
	back := fromCordext(x)
	if back.String() != c.String() {
		t.Fatalf("cordext->root bridge mismatch: root=%q back=%q", c.String(), back.String())
	}
	if back.Summary() != c.Summary() {
		t.Fatalf("summary mismatch after roundtrip: root=%+v back=%+v", c.Summary(), back.Summary())
	}
}

func TestBridgeCordextOperationParity(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	c := FromString("abcdef")
	x := toCordext(c)

	var err error
	c, err = Insert(c, FromString("X"), 3)
	if err != nil {
		t.Fatalf("root insert failed: %v", err)
	}
	x, err = x.Insert(toCordext(FromString("X")), 3)
	if err != nil {
		t.Fatalf("cordext insert failed: %v", err)
	}
	back := fromCordext(x)
	if back.String() != c.String() {
		t.Fatalf("insert bridge mismatch: root=%q back=%q", c.String(), back.String())
	}
}

func TestBridgeCordextEmpty(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	var c Cord
	x := toCordext(c)
	if !x.IsVoid() {
		t.Fatalf("expected void cordext value for empty root cord")
	}
	back := fromCordext(x)
	if !back.IsVoid() {
		t.Fatalf("expected void root cord after empty roundtrip")
	}
}
