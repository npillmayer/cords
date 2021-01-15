package metrics

import (
	"fmt"
	"regexp"

	"github.com/npillmayer/cords"
)

// --- Line count metric -----------------------------------------------------

// lineCount is a cords.Metric that counts the lines of a text, delimited by newline
// characters.
type lineCount struct {
	delimiterMetric
}

// LineCount is a cords.Metric that counts the lines of a text, delimited by newline
// characters. Multiple consecutive newlines will be counted as multiple empty lines.
// Clients who have a need for interpreting consecutive newlines in a different way
// may use a ParagraphCount metric first.
func LineCount() cords.Metric {
	m, _ := makeDelimiterMetric("\n", 1)
	return m
}

// Apply counts the lines in a text fragment.
// Apply is part of interface cords.Metric.
func (cnt *lineCount) Apply(frag string) cords.MetricValue {
	v := &linesCounted{}
	v.InitFrom(frag)
	v.Measured(0, len(frag), frag) // matches are of length 1, therefore not unprocessed bytes
	return v
}

// linesCounted is a cords.MetricValue
type linesCounted struct {
	delimiterMetricValue
}

// TODO do something smart with newline at end of text
func (cnt *lineCount) Combine(leftSibling, rightSibling cords.MetricValue,
	metric cords.Metric) cords.MetricValue {
	//
	l, r := leftSibling.(*linesCounted), rightSibling.(*linesCounted)
	if unproc, ok := l.ConcatUnprocessed(&r.MetricValueBase); ok {
		metric.Apply(string(unproc)) // we will not have unprocessed boundary bytes
	}
	l.UnifyWith(&r.MetricValueBase)
	return l
}

func (lc linesCounted) Count() int {
	return len(lc.parts) + 1
}

// var _ standardValue = linesCounted{}

// CountOf returns the count value of a given MetricValue, which must have been
// calculated from one of the metrics from this package (`metrics`).
//
// A return value of -1 flags that an unknown metrics value type has been given.
//
func CountOf(v cords.MetricValue) int {
	if d, ok := v.(*delimiterMetricValue); ok {
		// TODO this is not what we want
		// we need a layer in between
		return len(d.parts)
	}
	T().Errorf("metrics.CountOf called with unknown metric value type")
	return -1
}

// type standardValue interface {
// 	Count() int
// }

// --- Delimiter Metric ------------------------------------------------------

type delimiterMetric struct {
	pattern     *regexp.Regexp
	maxMatchLen int
}

type delimiterMetricValue struct {
	cords.MetricValueBase
	parts [][]int // int instead of int64 because of package regexp API
}

// A value of 0 for maxlen flags that the client cannot provide a maximum length
// of a match.
func makeDelimiterMetric(pattern string, maxlen int) (*delimiterMetric, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		T().Errorf("delimiter metric: cannot compile regular expression input")
		return nil, fmt.Errorf("illegal delimiter: %w", err)
	}
	if r.MatchString("") {
		T().Errorf("delimiter metric: regular expression matches empty string")
		return nil, cords.ErrIllegalDelimiterPattern
	}
	if maxlen < 0 {
		maxlen = 0
	}
	return &delimiterMetric{pattern: r, maxMatchLen: maxlen}, nil
}

func (dm *delimiterMetric) Apply(frag string) cords.MetricValue {
	v := delimiterMetricValue{}
	v.InitFrom(frag)
	v.parts = delimit(frag, dm.pattern)
	if len(v.parts) == 0 {
		v.MeasuredNothing(frag)
		// TODO should check if |v.parts|==1 and v.parts.0 @ 0
		// this could mean a pattern has started in the left sibling
		// with patterns like 'x+'
		// TODO respect maxMatchLen for this: e.g., if max = 4 and
		// a match at 0 has length 4, then not unprocessed boundary bytes
		// remain and reconsulting is unecessary.
	} else {
		v.Measured(v.parts[0][0], v.parts[len(v.parts)-1][1], frag)
	}
	return &v
}

func (dm *delimiterMetric) Combine(leftSibling, rightSibling cords.MetricValue,
	metric cords.Metric) cords.MetricValue {
	//
	l, ok := leftSibling.(*delimiterMetricValue)
	if !ok {
		T().Errorf("metric calculation: type of value is %T", leftSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	r, ok := rightSibling.(*delimiterMetricValue)
	if !ok {
		T().Errorf("metric calculation: type of value is %T", rightSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	if unproc, ok := l.ConcatUnprocessed(&r.MetricValueBase); ok {
		if d := metric.Apply(string(unproc)).(*delimiterMetricValue); len(d.parts) > 0 {
			l.parts = append(l.parts, d.parts...)
		}
	}
	l.UnifyWith(&r.MetricValueBase)
	l.parts = append(l.parts, r.parts...)
	return l
}

func (v *delimiterMetricValue) String() string {
	openL, openR := v.Unprocessed()
	return fmt.Sprintf("value{ length=%d, L='%s', R='%s', |P|=%d  }", v.Len(),
		string(openL), string(openR), len(v.parts))
}

func delimit(frag string, pattern *regexp.Regexp) (parts [][]int) {
	parts = pattern.FindAllStringIndex(frag, -1)
	if len(parts) == 0 {
		parts = [][]int{} // no boundary in fragment
	}
	return
}
