package cords

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestMetricBasic(t *testing.T) {
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
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
	metric := &testmetric{}
	v, err := ApplyMetric(cord, 0, cord.Len(), metric)
	if err != nil {
		t.Fatalf("application of test metric returned error: %v", err.Error())
	}
	t.Logf("metric value = %v", v)
	if v.Len() != 22 {
		t.Errorf("expected metric value of 22, have %d", v.Len())
	}
}

// --- Test helpers ----------------------------------------------------------

type testmetric struct{}

type testvalue struct {
	MetricValueBase
}

func (v *testvalue) Combine(rightSibling MetricValue, metric Metric) MetricValue {
	sibling := rightSibling.(*testvalue)
	if unproc, ok := v.ConcatUnprocessed(&sibling.MetricValueBase); ok {
		metric.Apply(string(unproc)) // we will not have unprocessed boundary bytes
	}
	v.UnifyWith(&sibling.MetricValueBase)
	return v
}

func (m *testmetric) Apply(frag string) MetricValue {
	v := &testvalue{}
	v.InitFrom(frag)
	v.Measured(0, len(frag), frag) // test metric simply counts bytes
	return v
}
