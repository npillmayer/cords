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
	if c.root.height != 2 {
		t.Errorf("Height of root = %d, should be 2", c.root.height)
	}
	leaf := c.root.left
	if !leaf.IsLeaf() {
		t.Errorf("expected leaf at height 1, is not")
	}
	// t.Logf("parent=%v", leaf.parent)
	// if leaf.parent.left != leaf || leaf.parent.right != nil {
	// 	t.Errorf("root node not constructed as expected")
	// }
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
	if node.String()[i] != 'W' || i != 6 {
		t.Error("expected index at 6/'W', isn't")
	}
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
	t.Logf("c.right = '%s'", c.root.Right())
	if c.root.height != 3 {
		t.Errorf("expected height(c) to be 3, is %d", c.root.Height())
	}
	// mid := c.root.left.AsNode()
	// t.Logf("mid = %v", mid)
	// if mid.left.parent != mid || mid.right.parent != mid {
	// 	t.Logf("mid.left.parent  = %v", mid.left.parent)
	// 	t.Logf("mid.right.parent = %v", mid.right.parent)
	// 	t.Errorf("cord structure not as expected")
	// }
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
	if node.String()[i] != 'W' || i != 0 {
		t.Error("expected index at 0/'W', isn't")
	}
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

// func TestUnzip(t *testing.T) {
// 	gtrace.CoreTracer = gotestingadapter.New()
// 	teardown := gotestingadapter.RedirectTracing(t)
// 	defer teardown()
// 	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
// 	//
// 	c1 := FromString("Hello ")
// 	c2 := FromString("World")
// 	c := Concat(c1, c2)
// 	leaf, i, err := c.index(7)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	t.Logf("str[%d] = %c", i, leaf.String()[i])
// 	top := unzip(&leaf.cordNode, c.root)
// 	if top == nil {
// 		t.Fatal("top is nil, should be clone of c.root")
// 	}
// 	dump(&top.cordNode)
// 	if top == c.root {
// 		t.Fatal("top = c.root, should be clone of c.root")
// 	}
// }

func TestCordSplit1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2)
	cl, cr, err := Split(c, 2)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("----------------------")
	dump(&cl.root.cordNode)
	t.Logf("----------------------")
	dump(&cr.root.cordNode)
	t.Logf("======================")
	dump(&c.root.cordNode)
	t.Fail()
}
