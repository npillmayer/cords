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
	if &c1.root.cordNode == c.root.left {
		t.Errorf("copy on write did not work for c1.root")
	}
}

func TestCordLength1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello ")
	t.Logf("c1.len=%d", c1.root.Len())
	c2 := FromString("World")
	t.Logf("c2.len=%d", c2.root.Len())
	c := Concat(c1, c2)
	if c.root.Len() != c.Len() {
		t.Errorf("length calculation of top inner node failed, %d != %d", c.root.Len(), c.Len())
	}
	if c.root.left.Len() != c.Len() {
		t.Logf("w=%d", c.root.left.Weight())
		t.Errorf("length calculation of inner node failed, %d != %d", c.root.left.Len(), c.Len())
	}
	if c.root.Len() != 11 || c.root.Left().Len() != 11 {
		t.Errorf("length calculation is off, expected 11, is %d|%d", c.root.Len(), c.root.Left().Len())
	}
}

func TestRotateLeft(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello")
	c2 := FromString(" World,")
	c3 := FromString(", how are you?")
	c := Concat(c1, c2)
	dump(&c.root.cordNode)
	t.Logf("-----------------------------------------------")
	c = Concat(c, c3)
	dump(&c.root.cordNode)
	t.Logf("-----------------------------------------------")
	x := rotateRight(c.root)
	dump(&x.cordNode)
	if x.Left().Height() != 2 || x.Right().Height() != 2 {
		t.Errorf("Expected both left and right sub-tree to be of height 2, aren't")
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
	b := !unbalanced(c.root.left)
	t.Logf("balance of c = %v", b)
	t.Logf("height of left = %d", c.root.Left().Height())
	if !b || c.root.Left().Height() != 3 {
		dump(&c.root.cordNode)
		t.Fail()
	}
}

func TestCordSplit1(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2) // now have a tree of height 3
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
	if cl.root == nil || cr.root == nil {
		t.Fatalf("Split resulted in empty partial cord, should not")
	}
	if cl.root.Height() != 2 || cr.root.Height() != 3 {
		t.Errorf("Expected split sub-trees of height 2 and 3, are %d and %d", cl.root.Height(), cr.root.Height())
	}
	if cl.String() != "He" || cr.String() != "llo World" {
		t.Errorf("Expected split 'He'|'llo World', but left part is %v", cl)
	}
	if c.root == cl.root || c.root == cr.root {
		t.Fatalf("copy on write did not work as expected")
	}
}

func TestCordInsert(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2) // Hello World
	x, err := Insert(c, FromString(","), 5)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("x = '%s'", x.String())
	if x.Len() != 12 {
		t.Errorf("Expected result to be of length 12, is %d", x.Len())
	}
	x, err = Insert(x, FromString("!"), 12)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("x = '%s'", x.String())
	if x.String() != "Hello, World!" {
		t.Errorf("Double insert resulted in inexpected string: %s", x)
	}
}

func TestCordDelete(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2)       // Hello World
	x, err := Delete(c, 4, 4) // => Hellrld
	if err != nil {
		t.Fatalf(err.Error())
	}
	if x.String() != "Hellrld" {
		t.Errorf("Expected delete-result to be 'Hellrld', is '%s'", x)
	}
}

func TestCordReport(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c := FromString("Hello_")
	c = Concat(c, FromString("my_"))
	c = Concat(c, FromString("na"))
	c = Concat(c, FromString("me_i"))
	c = Concat(c, FromString("s"))
	c = Concat(c, FromString("_Simon"))
	t.Logf("cord='%s'", c)
	s, err := c.Report(8, 5)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("s='%s'", s)
	if s != "_name" {
		t.Errorf("Expected resulting string to be '_name', is '%s'", s)
	}
}
