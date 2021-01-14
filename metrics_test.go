package cords

import (
	"regexp"
	"testing"

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
	t.Fail()
}

func TestDelimit(t *testing.T) {
	content := "Londonderry"
	pattern := `o.`
	t.Logf("delimit '%s' by /%s/", content, pattern)
	re := regexp.MustCompile(`o.`)
	m := delimit(content, re)
	t.Logf("m=%v", m)
	t.Fail()
}

func TestMetricBasic(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	b := NewBuilder()
	b.Append(StringLeaf("name_is"))
	b.Prepend(StringLeaf("Hello_my_"))
	b.Append(StringLeaf("_Simon"))
	cord := b.Cord()
	if cord.IsVoid() {
		t.Fatalf("Expected non-void result cord, is void")
	}
	dump(&cord.root.cordNode)
	t.Logf("builder made cord='%s'", cord)
	metric, err := makeDelimiterMetric("_")
	if err != nil {
		t.Errorf(err.Error())
	}
	value := &delimiterMetricValue{length: 0}
	v := applyMetric(&cord.root.cordNode, 0, cord.Len(), metric, value)
	t.Logf("delimiter value = %v", v)
	t.Fail()
}
