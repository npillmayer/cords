package metrics

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestRegexAllSubmatchIndex(t *testing.T) {
	content := []byte(`
	# comment line
	option1: value1
	option2: value2
`)
	// Regex pattern captures "key: value" pair from the content.
	pattern := regexp.MustCompile(`(?m)(?P<key>\w+):\s+(?P<value>\w+)$`)
	allIndexes := pattern.FindAllSubmatchIndex(content, -1)
	for _, loc := range allIndexes {
		t.Logf("loc=%v", loc)
		t.Logf(string(content[loc[0]:loc[1]]))
		t.Logf(string(content[loc[2]:loc[3]]))
		t.Logf(string(content[loc[4]:loc[5]]))
	}
	//t.Fail()
}

func TestRegexAllIndex(t *testing.T) {
	content := []byte("on")
	pattern := `o.`
	t.Logf("search /%s/ in '%s'", pattern, string(content))
	re := regexp.MustCompile(`o.`)
	m := re.FindAllIndex(content, -1)
	t.Logf("intervals found: %v", m)
	if len(m) > 0 && m[0][0] >= 0 {
		t.Logf("%d matches, first match = '%s'", len(m), string(content[m[0][0]:m[0][1]]))
	} else {
		t.Logf("no match")
	}
	//t.Fail()
}

func TestDelimit(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	content := []byte("Londonderry")
	pattern := `o.`
	t.Logf("delimit '%s' by /%s/", content, pattern)
	re := regexp.MustCompile(`o.`)
	m := delimit(content, re)
	t.Logf("m=%v", m)
	if len(m) != 2 {
		t.Errorf("expected to find 2 pattern occurences, found %d", len(m))
	}
}

func TestMetricLines(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	b := cords.NewBuilder()
	b.Append(cords.StringLeaf("Hello\n"))
	b.Append(cords.StringLeaf("my\n"))
	b.Append(cords.StringLeaf("name\n"))
	b.Append(cords.StringLeaf("is\n"))
	b.Append(cords.StringLeaf("Simon"))
	cord := b.Cord()
	if cord.IsVoid() {
		t.Fatalf("Expected non-void result cord, is void")
	}
	t.Logf("builder made cord='%s'", cord)
	//
	t.Logf("--- count --------------------------------")
	cnt, err := Count(cord, 0, cord.Len(), LineCount())
	if err != nil {
		t.Fatalf(err.Error())
	}
	if cnt != 4 {
		t.Errorf("expected to find 5 lines, found %d", cnt)
	}
	//
	t.Logf("--- find ---------------------------------")
	locs, err := Find(cord, 0, cord.Len(), FindLines())
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(locs) != 4 {
		for _, loc := range locs {
			t.Logf("line at %d with length %d", loc[0], loc[1])
		}
		t.Errorf("expected to find 4 '\\n'-terminated lines, found %d", len(locs))
	}
}

func TestSplitFunc(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	str := strings.NewReader("")
	//str := strings.NewReader("the quick brown fox")
	scnr := bufio.NewScanner(str)
	scnr.Split(splitWords)
	cnt := 0
	for scnr.Scan() {
		t.Logf("t='%s'", scnr.Text())
		cnt++
	}
	if cnt != 7 {
		t.Errorf("expected 7 tokens, got %d", cnt)
	}
}

func TestMetricSpanWord(t *testing.T) {
	// gtrace.CoreTracer = gotestingadapter.New()
	// teardown := gotestingadapter.RedirectTracing(t)
	// defer teardown()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	text := cords.FromString("Hello my name is Simon")
	//
	metric := Words()
	value, cord, err := Align(text, 0, text.Len(), metric)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("value of materialized metric = %v", value)
	if cord.IsVoid() {
		t.Fatalf("resulting aligned cord is void, shouldn't")
	}
	tmpfile := dotty(cord, t)
	defer tmpfile.Close()
}

func TestMetricWordSpans(t *testing.T) {
	// gtrace.CoreTracer = gotestingadapter.New()
	// teardown := gotestingadapter.RedirectTracing(t)
	// defer teardown()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	b := cords.NewBuilder()
	b.Append(cords.StringLeaf("Hello "))
	b.Append(cords.StringLeaf("my "))
	b.Append(cords.StringLeaf("na"))
	b.Append(cords.StringLeaf("me i"))
	b.Append(cords.StringLeaf("s"))
	b.Append(cords.StringLeaf(" Simon"))
	text := b.Cord()
	if text.IsVoid() {
		t.Fatalf("Expected non-void result cord, is void")
	}
	t.Logf("builder made cord='%s'", text)
	tmpfile := dotty(text, t)
	defer tmpfile.Close()
	//
	t.Fail()
	metric := Words()
	value, cord, err := Align(text, 0, text.Len(), metric)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("value of materialized metric = %v", value)
	if cord.IsVoid() {
		t.Fatalf("resulting aligned cord is void, shouldn't")
	}
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
