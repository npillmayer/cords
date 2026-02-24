package btree

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestPipelineSimple(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	tree := buildTextTree(t, 60)
	var chunk textChunk
	p := pipeFor(tree)
	p.err = p.call(p.tree.Check)
	t.Logf("error: %v", p.err)
	p.tree, p.err = pipeCall1(p, p.tree.DeleteAt, 3)
	p.item, p.err = pipeCall1(p, p.tree.At, 3)
	t.Logf("chunk: %v", chunk)
	c := p.itemOrElse(textChunk("<void>"))
	t.Logf("chunk: %v", c)
	if c.String() != "4" {
		t.Fatal("pipeline failed")
	}
}
