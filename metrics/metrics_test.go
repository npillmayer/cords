package metrics

import (
	"regexp"
	"testing"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
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
	content := "Londonderry"
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
	metric := LineCount()
	cnt, err := cords.ApplyMetric(cord, 0, cord.Len(), metric)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("count value = %v", cnt)
	if CountOf(cnt) != 4 {
		t.Errorf("expected to find 5 lines, found %d", CountOf(cnt))
	}
}
