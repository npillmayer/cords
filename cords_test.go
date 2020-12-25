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
	gtrace.SyntaxTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c := FromString("Hello World")
	t.Logf("c = '%s'", c)
	if c.String() != "Hello World" {
		t.Errorf("Expected cords.String() to be 'Hello World', is not")
	}
}

func TestCordIndex(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.SyntaxTracer.SetTraceLevel(tracing.LevelDebug)
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
	gtrace.SyntaxTracer.SetTraceLevel(tracing.LevelDebug)
	//
	c1 := FromString("Hello World")
	c2 := FromString(", how are you?")
	c := Concat(c1, c2)
	if c.IsVoid() {
		t.Fatalf("concatenation is nil")
	}
	t.Logf("c = '%s'", c)
	t.Logf("c.left = '%s'", c.root.Left().Left())
	t.Fail()
}
