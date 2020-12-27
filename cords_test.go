package cords

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestNewStringCord(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c := FromString("Hello World")
	t.Logf("c = '%s'", c)
	if c.String() != "Hello World" {
		t.Errorf("Expected cords.String() to be 'Hello World', is not")
	}
}

func TestCordIndex1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c := FromString("Hello World")
	node, i, err := c.index(6)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("str[%d] = %c", i, node.String()[i])
}

func TestCordConcat(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello World")
	c2 := FromString(", how are you?")
	c := Concat(c1, c2)
	if c.IsVoid() {
		t.Fatalf("concatenation is nil")
	}
	t.Logf("c = '%s'", c)
	t.Logf("c.left = '%s'", c.root.Left())
	if c.root.height != 3 {
		t.Errorf("expected height(c) to be 3, is %d", c.root.Height())
	}
	if c.root.left.String() != "<inner node>" {
		t.Errorf("cord structure differs from expected")
	}
	if &c1.root.cordNode == c.root.left {
		t.Errorf("copy on write did not work for c1.root")
	}
}

func TestCordIndex2(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2)
	node, i, err := c.index(6)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("str[%d] = %c", i, node.String()[i])
}

func TestBalance1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello")
	c2 := FromString(" World,")
	c3 := FromString(", how are")
	c4 := FromString("you?")
	c := Concat(c1, c2)
	c = Concat(c, c3)
	c = Concat(c, c4)
	b := !unbalanced(c.root)
	t.Logf("balance of c = %v", b)
	t.Logf("height of left = %d", c.root.Left().Height())
	if !b || c.root.Left().Height() != 3 {
		dump(&c.root.cordNode)
		t.Fail()
	}
}
