package metrics

import (
	"fmt"
	"regexp"

	"github.com/npillmayer/cords"
)

// --- Line count metric -----------------------------------------------------

// LineCount creates a CountingMetrc to be applied to a cord.
// It counts the lines of a text, delimited by newline characters.
// Multiple consecutive newlines will be counted as multiple empty lines.
// Clients who have a need for interpreting consecutive newlines in a different way
// may use a ParagraphCount metric first. If the text does not end with a newline,
// the trailing text fragment is *not* counted as a (incomplete) line.
func LineCount() CountingMetric {
	m, _ := makeDelimiterMetric("\n", 1)
	lcnt := &lineCountMetric{m}
	return lcnt
}

// lineCountMetric is a CountingMetric that counts the lines of a text, delimited
// by newline characters.
type lineCountMetric struct {
	*delimiterMetric
}

// Count returns a line count, previously calculated by application of metric `lcnt`,
// by decoding v.
//
// Count is part of interface CountingMetric
func (lcnt *lineCountMetric) Count(v cords.MetricValue) int {
	n, ok := v.(*delimiterMetricValue)
	if !ok {
		panic("metric value is not a counting metric value for lines")
	}
	return len(n.parts)
}

// Apply is part of interface cords.Metric.
func (lcnt *lineCountMetric) Apply(frag []byte) cords.MetricValue {
	return lcnt.delimiterMetric.Apply(frag)
}

// Combine is part of interface cords.Metric.
func (lcnt *lineCountMetric) Combine(leftSibling, rightSibling cords.MetricValue,
	metric cords.Metric) cords.MetricValue {
	//
	return lcnt.delimiterMetric.Combine(leftSibling, rightSibling, metric)
}

// ---------------------------------------------------------------------------

// FindLines creates a ScanningMetrc to be applied to a cord.
// It finds the lines of a text, delimited by newline characters.
// Multiple consecutive newlines will be counted as multiple empty lines.
// Clients who have a need for interpreting consecutive newlines in a different way
// may use a ParagraphCount metric first.
//
// FindLines returns tuples [position, length] for each line of text, not counting
// the line-terminating newline characters. If the last text fragment does not contain
// a final newline, it will be reported as a (fractional) line. Clients who have a
// need for the last non-terminated line will have to use cord.Report, starting at
// position+length+1 of the final location from FindLines.
func FindLines() ScanningMetric {
	m, _ := makeDelimiterMetric("\n", 1)
	fl := &findLinesMetric{m}
	return fl
}

type findLinesMetric struct {
	*delimiterMetric
}

// Indexes returns a positions of lines of text, previously calculated by application
// by decoding a delimiterMetricValue given as a a cords.MetricValue
//
// Locations is part of interface ScanningMetric
func (fl *findLinesMetric) Locations(v cords.MetricValue) [][]int {
	n, ok := v.(*delimiterMetricValue)
	if !ok {
		panic("metric value is not a delimiter metric value for locations of lines")
	}
	locs := make([][]int, len(n.parts))
	pos := 0
	for i, nl := range n.parts {
		tracer().Debugf("pos=%d, nl=%v", pos, nl)
		locs[i] = make([]int, 2)
		locs[i][0] = pos
		locs[i][1] = nl[0] - pos
		pos = nl[1]
	}
	return locs
}

// Apply is part of interface cords.Metric.
func (fl *findLinesMetric) Apply(frag []byte) cords.MetricValue {
	return fl.delimiterMetric.Apply(frag)
}

// Combine is part of interface cords.Metric.
func (fl *findLinesMetric) Combine(leftSibling, rightSibling cords.MetricValue,
	metric cords.Metric) cords.MetricValue {
	//
	return fl.delimiterMetric.Combine(leftSibling, rightSibling, metric)
}

// --- Delimiter Metric ------------------------------------------------------

type delimiterMetric struct {
	pattern     *regexp.Regexp
	maxMatchLen int
}

type delimiterMetricValue struct {
	MetricValueBase
	parts [][]int // int instead of int64 because of package regexp API
}

// A value of 0 for maxlen flags that the client cannot provide a maximum length
// of a match.
func makeDelimiterMetric(pattern string, maxlen int) (*delimiterMetric, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		tracer().Errorf("delimiter metric: cannot compile regular expression input")
		return nil, fmt.Errorf("illegal delimiter: %w", err)
	}
	if r.MatchString("") {
		tracer().Errorf("delimiter metric: regular expression matches empty string")
		return nil, cords.ErrIllegalDelimiterPattern
	}
	if maxlen < 0 {
		maxlen = 0
	}
	return &delimiterMetric{pattern: r, maxMatchLen: maxlen}, nil
}

func (dm *delimiterMetric) Apply(frag []byte) cords.MetricValue {
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
		tracer().Errorf("metric calculation: type of value is %T", leftSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	r, ok := rightSibling.(*delimiterMetricValue)
	if !ok {
		tracer().Errorf("metric calculation: type of value is %T", rightSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	if unproc, ok := l.ConcatUnprocessed(&r.MetricValueBase); ok {
		if d := metric.Apply(unproc).(*delimiterMetricValue); len(d.parts) > 0 {
			l.parts = append(l.parts, d.parts...)
		}
	}
	offset := l.Len()
	l.UnifyWith(&r.MetricValueBase)
	l.parts = combine(l.parts, r.parts, offset)
	return l
}

func (v *delimiterMetricValue) String() string {
	openL, openR := v.Unprocessed()
	return fmt.Sprintf("value{ length=%d, L='%s', R='%s', |P|=%d  }", v.Len(),
		string(openL), string(openR), len(v.parts))
}

func delimit(frag []byte, pattern *regexp.Regexp) (parts [][]int) {
	parts = pattern.FindAllIndex(frag, -1)
	if len(parts) == 0 {
		parts = [][]int{} // no boundary in fragment
	}
	return
}

func combine(l, r [][]int, offset int) [][]int {
	tracer().Debugf("range %v ++ %v , offset=%d", l, r, offset)
	for _, p := range r {
		p[0], p[1] = offset+p[0], offset+p[1]
		l = append(l, p)
	}
	tracer().Debugf("combined index ranges = %v", l)
	return l
}
