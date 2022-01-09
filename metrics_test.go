package cords

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestDotty(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	b := NewBuilder()
	b.Append(StringLeaf("name_is"))
	b.Prepend(StringLeaf("Hello_my_"))
	b.Append(StringLeaf("_Simon"))
	text := b.Cord()
	if text.IsVoid() {
		t.Fatalf("Expected non-void result cord, is void")
	}
	dump(&text.root.cordNode)
	t.Logf("builder made cord='%s'", text)
	// tmpfile := dotty(text, t)
	// defer tmpfile.Close()
}

// --- dot -------------------------------------------------------------------

func dotty(text Cord, t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile(".", "cord.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing digraph to %s\n", tmpfile.Name())
	Cord2Dot(text, tmpfile)
	cmd := exec.Command("dot", "-Tsvg", "-otree.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG tree image to tree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
	return tmpfile
}
