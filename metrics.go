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
}

type delimiterMetric struct {
	pattern *regexp.Regexp
}

type delimiterMetricValue struct {
	length       int64   //
	openL, openR []byte  //
	parts        [][]int // int instead of int64 because of package regexp API
}

func makeDelimiterMetric(pattern string) (*delimiterMetric, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		T().Debugf("delimiter metric: cannot compile regular expression input")
		return nil, fmt.Errorf("delimiter metric: illegal delimiter: %w", err)
	}
	return &delimiterMetric{pattern: r}, nil
}

func (dm *delimiterMetric) Apply(frag string) MetricValue {
	v := delimiterMetricValue{
		length: int64(len(frag)),
	}
	v.parts = delimit(frag, dm.pattern)
	if len(v.parts) == 0 {
		v.openL = []byte(frag)
	}
	if len(v.parts) > 0 {
		if v.parts[0][0] > 0 {
			v.openL = []byte(frag)[:v.parts[0][0]]
		}
	}
	if len(v.parts) > 0 {
		l := len(v.parts)
		if v.parts[l-1][1] < int(v.length) {
			v.openR = []byte(frag)[v.parts[l-1][1]:]
		}
	}
	return &v
}

func (v *delimiterMetricValue) String() string {
	return fmt.Sprintf("value{ length=%d, L='%s', R='%s', |P|=%d  }", v.length,
		string(v.openL), string(v.openR), len(v.parts))
}

func (v *delimiterMetricValue) Combine(rightSibling MetricValue, metric Metric) MetricValue {
	sibling, ok := rightSibling.(*delimiterMetricValue)
	if !ok {
		T().Errorf("metric calculation: type of value is %T", rightSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	T().Debugf("COMBINE %v  +  %v", v, sibling)
	if v.length == 0 {
		return sibling
	}
	if sibling.length == 0 {
		return v
	}
	vv := delimiterMetricValue{
		length: v.length + sibling.length,
		parts:  v.parts,
	}
	l := len(v.parts)
	if l == 0 { // we have a span without boundaries
		if len(sibling.openL) > 0 {
			// append sibling.openL to it
			vv.openL = append(v.openL, sibling.openL...)
			vv.openR = sibling.openR
		} else {
			// we just use v as is
			vv.openL = v.openL
			vv.openR = sibling.openR
		}
	} else {
		// we have at least 1 boundary
		if len(v.openL) > 0 {
			// we do nothing here, for sibling is to our right
			vv.openL = v.openL
		}
		if len(v.openR) > 0 {
			// append sibling.openL, if any, and apply metric
			if len(sibling.openL) > 0 {
				x := append(v.openL, sibling.openL...)
				vx := metric.Apply(string(x))
				vvx := vx.(*delimiterMetricValue)
				if len(vvx.parts) > 0 {
					// append newfound boundary
					vv.parts = append(vv.parts, vvx.parts...)
					vv.openR = vvx.openR
				} else {
					vv.openR = sibling.openR
				}
			}
		} else {
			// we already have tested v.openR, so we do nothing here
			vv.openR = sibling.openR
		}
	}
	vv.parts = append(vv.parts, sibling.parts...)
	return &vv
}

func delimit(frag string, pattern *regexp.Regexp) (parts [][]int) {
	parts = pattern.FindAllStringIndex(frag, -1)
	if len(parts) == 0 {
		parts = [][]int{} // no boundary in fragment
	}
	// parts := make([]int64, len(m), len(m))
	// for i, pos := range m {
	// 	parts[i] = int64(pos[0])
	// }
	return
}

func applyMetric(node *cordNode, i, j uint64, metric Metric, value MetricValue) MetricValue {
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
		vl = applyMetric(node.Left(), i, j, metric, value)
		T().Debugf("left metric value = %v", vl)
	}
	if node.Right() != nil && j > node.Weight() {
		w := node.Weight()
		vr = applyMetric(node.Right(), i-umin(w, i), j-w, metric, value)
		T().Debugf("right metric value = %v", vr)
	}
	if vl != nil && vr != nil {
		v = vl.Combine(vr, metric) // TODO we should copy/clone
	} else if vl != nil {
		v = vl
	} else if vr != nil {
		v = vr
	}
	T().Debugf("combined metric value = %v", v)
	T().Debugf("node=%v", node)
	T().Debugf("dropping out of applyMetric([%d], %d, %d)", node.Weight(), i, j)
	return v
}
