package html

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"golang.org/x/net/html"
)

func TestHTMLSimple(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	r := strings.NewReader(`
	<!DOCTYPE html>
	<html>
	<body>
	
	<h1>My First Heading</h1>
	<p>My <b>first</b> paragraph.</p>
	
	</body>
	</html> 
`)
	doc, err := html.Parse(r)
	if err != nil {
		t.Fatalf(err.Error())
	}
	text, err := InnerText(doc)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("text = '%s'", text)
	tmpfile := dotty(text, t)
	defer tmpfile.Close()
	//t.Fail()
}

func TestHTMLParse(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	r := strings.NewReader(`<p>My <b>first</b> paragraph.</p>`)
	text, err := TextFromHTML(r)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("text = '%s'", text)
	tmpfile := dotty(text, t)
	defer tmpfile.Close()
	//t.Fail()
}

// --- Helpers ---------------------------------------------------------------

func dotty(text cords.Cord, t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile(".", "cord.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing digraph to %s\n", tmpfile.Name())
	cords.Cord2Dot(text, tmpfile)
	cmd := exec.Command("dot", "-Tsvg", "-otree.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG tree image to tree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
	return tmpfile
}
