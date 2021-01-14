package cords

import (
	"fmt"
	"regexp"
)

type Metric interface {
	//Leafs(leftMetric Metric) []Leaf
	Apply(frag string) MetricValue
	//Value() MetricValue
}

type MetricValue interface {
	Combine(rightSibling MetricValue, metric Metric) MetricValue
	Unprocessed() ([]byte, []byte)
	Len() int
}

type MetricValueBase struct {
	length       int
	openL, openR []byte //
}

func (mvb MetricValueBase) Len() int {
	return mvb.length
}

func (mvb MetricValueBase) Unprocessed() ([]byte, []byte) {
	return mvb.openL, mvb.openR
}

func (mvb *MetricValueBase) InitFrom(frag string) {
	mvb.length = len(frag)
}

func (mvb *MetricValueBase) Measured(from, to int, frag string) {
	if from < 0 || from > mvb.length {
		mvb.openL = []byte(frag)
		mvb.openR = nil
		return
	}
	mvb.openL = nil
	mvb.openR = nil
	if from > to {
		from, to = to, from
	}
	if from > 0 {
		mvb.openL = []byte(frag)[:from]
	}
	if to < mvb.length {
		mvb.openR = []byte(frag)[to:]
	}
}

func (mvb *MetricValueBase) MeasuredNothing(frag string) {
	mvb.Measured(-1, -1, frag)
}

func (mvb *MetricValueBase) HasBoundaries() bool {
	return len(mvb.openL)+len(mvb.openR) < mvb.length
}

// will change mvb
func (mvb *MetricValueBase) ConcatUnprocessed(rightSibling *MetricValueBase) ([]byte, bool) {
	otherL := rightSibling.openL
	if len(otherL) > 0 {
		if mvb.HasBoundaries() {
			mvb.openR = append(mvb.openR, otherL...)
			return mvb.openR, rightSibling.HasBoundaries()
		}
		// else no boundaries in mvb => openR is empty, frag is in openL
		mvb.openL = append(mvb.openL, otherL...)
		return mvb.openL, false
	}
	return nil, false
}

func (mvb *MetricValueBase) UnifyWith(rightSibling *MetricValueBase) {
	mvb.length += rightSibling.length
	mvb.openR = rightSibling.openR
}

// --- Delimiter Metric ------------------------------------------------------

type delimiterMetric struct {
	pattern *regexp.Regexp
}

type delimiterMetricValue struct {
	MetricValueBase
	parts [][]int // int instead of int64 because of package regexp API
}

func makeDelimiterMetric(pattern string) (*delimiterMetric, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		T().Errorf("delimiter metric: cannot compile regular expression input")
		return nil, fmt.Errorf("illegal delimiter: %w", err)
	}
	if r.MatchString("") {
		T().Errorf("delimiter metric: regular expression matches empty string")
		return nil, ErrIllegalDelimiterPattern
	}
	return &delimiterMetric{pattern: r}, nil
}

func (dm *delimiterMetric) Apply(frag string) MetricValue {
	v := delimiterMetricValue{}
	v.InitFrom(frag)
	v.parts = delimit(frag, dm.pattern)
	if len(v.parts) == 0 {
		v.MeasuredNothing(frag)
		// TODO should check if |v.parts|==1 and v.parts.0 @ 0
		// this could mean a pattern has started in the left sibling
		// with patterns like 'x+'
	} else {
		v.Measured(v.parts[0][0], v.parts[len(v.parts)-1][1], frag)
	}
	return &v
}

func (v *delimiterMetricValue) Combine(rightSibling MetricValue, metric Metric) MetricValue {
	sibling, ok := rightSibling.(*delimiterMetricValue)
	if !ok {
		T().Errorf("metric calculation: type of value is %T", rightSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	if unproc, ok := v.ConcatUnprocessed(&sibling.MetricValueBase); ok {
		if d := metric.Apply(string(unproc)).(*delimiterMetricValue); len(d.parts) > 0 {
			v.parts = append(v.parts, d.parts...)
		}
	}
	v.UnifyWith(&sibling.MetricValueBase)
	v.parts = append(v.parts, sibling.parts...)
	return v
}

func (v *delimiterMetricValue) String() string {
	return fmt.Sprintf("value{ length=%d, L='%s', R='%s', |P|=%d  }", v.length,
		string(v.openL), string(v.openR), len(v.parts))
}

func delimit(frag string, pattern *regexp.Regexp) (parts [][]int) {
	parts = pattern.FindAllStringIndex(frag, -1)
	if len(parts) == 0 {
		parts = [][]int{} // no boundary in fragment
	}
	return
}

func applyMetric(node *cordNode, i, j uint64, metric Metric) MetricValue {
	T().Debugf("called applyMetric([%d], %d, %d)", node.Weight(), i, j)
	if node.IsLeaf() {
		leaf := node.AsLeaf()
		T().Debugf("METRIC(%s|%d, %d, %d)", leaf, leaf.Len(), i, j)
		s := leaf.leaf.Substring(umax(0, i), umin(j, leaf.Len()))
		v := metric.Apply(s)
		T().Debugf("leaf metric value = %v", v)
		return v
	}
	var v, vl, vr MetricValue
	if i < node.Weight() && node.Left() != nil {
		vl = applyMetric(node.Left(), i, j, metric)
		T().Debugf("left metric value = %v", vl)
	}
	if node.Right() != nil && j > node.Weight() {
		w := node.Weight()
		vr = applyMetric(node.Right(), i-umin(w, i), j-w, metric)
		T().Debugf("right metric value = %v", vr)
	}
	if !isnull(vl) && !isnull(vr) {
		T().Debugf("COMBINE %v  +  %v", vl, vr)
		v = vl.Combine(vr, metric) // TODO we should copy/clone
	} else if !isnull(vl) {
		v = vl
	} else if !isnull(vr) {
		v = vr
	}
	T().Debugf("combined metric value = %v", v)
	T().Debugf("node=%v", node)
	T().Debugf("dropping out of applyMetric([%d], %d, %d)", node.Weight(), i, j)
	return v
}

func isnull(v MetricValue) bool {
	return v == nil || v.Len() == 0
}
