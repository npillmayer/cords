package metrics

import (
	"bufio"
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/npillmayer/cords"
)

func splitWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	spaces := false
	pos := 0
	if data[0] < utf8.RuneSelf { // is ASCII
		if unicode.IsSpace(rune(data[0])) {
			spaces = true
		}
		pos = 1
	} else {
		r, width := utf8.DecodeRune(data)
		if unicode.IsSpace(r) {
			spaces = true
		}
		pos = width
	}
	//T().Debugf("spaces=%v", spaces)
	for {
		r, width := utf8.DecodeRune(data[pos:])
		//T().Debugf("r=%v, width=%d, pos=%d", r, width, pos)
		if r == utf8.RuneError {
			tracer().Debugf("rune error. atEOF=%v, width=%d", atEOF, width)
			if width == 0 {
				return pos, data[0:pos], nil
			}
			// Is the error because there wasn't a full rune to be decoded?
			// FullRune distinguishes correctly between erroneous and incomplete encodings.
			if !atEOF && !utf8.FullRune(data) {
				// Incomplete; get more bytes.
				return 0, nil, nil
			}
			// We have a real UTF-8 encoding error. Return a properly encoded error rune
			// but advance only one byte. This matches the behavior of a range loop over
			// an incorrectly encoded string.
			return 1, errorRune, nil
		}
		// It's a valid encoding. Width cannot be 1 for a correctly encoded non-ASCII rune.
		if spaces {
			if unicode.IsSpace(r) {
				advance += width
			} else {
				return pos, data[0:pos], nil
			}
		} else {
			if unicode.IsSpace(r) {
				return pos, data[0:pos], nil
			}
			advance += width
		}
		pos += width
	}
}

var errorRune = []byte(string(utf8.RuneError))

func Words() cords.MaterializedMetric {
	m, _ := makeSpanningMetric(wordScannerFactory)
	return m
}

func wordScannerFactory(input []byte) *bufio.Scanner {
	str := bytes.NewReader(input)
	scnr := bufio.NewScanner(str)
	scnr.Split(splitWords)
	return scnr
}

// --- Delimiter Metric ------------------------------------------------------

type spanningMetric struct {
	scnr func([]byte) *bufio.Scanner
}

type spanningMetricValue struct {
	MetricValueBase
	spans   [][]int // (pos,len); int instead of int64 because of package regexp API
	split   int     // signals that no span has been recognized, but a metric boundary
	lasterr error   // collect errors and preserve the last one
	mid     [][]int // mid section of spans to convert to leafs
}

func makeSpanningMetric(scnrFactory func([]byte) *bufio.Scanner) (*spanningMetric, error) {
	if scnrFactory == nil {
		return nil, cords.ErrIllegalArguments
	}
	return &spanningMetric{scnr: scnrFactory}, nil
}

func (sm *spanningMetric) Apply(frag []byte) cords.MetricValue {
	v := &spanningMetricValue{}
	v.InitFrom(frag)
	v = sm.scan([]byte(frag), v)
	tracer().Debugf("scan of '%s' returned %v", string(frag), v)
	if v.split > 0 {
		v.Measured(v.split, v.split, frag)
		//} else if !v.HasBoundaries() {
	} else if len(v.spans) == 0 {
		v.MeasuredNothing(frag)
	} else {
		v.Measured(v.spans[0][0], lastpos(v.spans), frag)
		v.mid = v.spans
	}
	return v
}

// Combine for materialzed metrics focuses on the unprocessed bytes of
// left.suffix and right.prefix, and have remember those for converting them
// to cord leafs.
func (sm *spanningMetric) Combine(leftSibling, rightSibling cords.MetricValue,
	metric cords.Metric) cords.MetricValue {
	//
	l, ok := leftSibling.(*spanningMetricValue)
	if !ok {
		tracer().Errorf("metric calculation: type of value is %T", leftSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	r, ok := rightSibling.(*spanningMetricValue)
	if !ok {
		tracer().Errorf("metric calculation: type of value is %T", rightSibling)
		panic("cords.Metric combine: type inconsistency in metric calculation")
	}
	l.mid = [][]int{}
	if unproc, ok := l.ConcatUnprocessed(&r.MetricValueBase); ok {
		d := metric.Apply(unproc).(*spanningMetricValue)
		if len(d.Suffix()) > 0 && l.HasBoundaries() {
			// make span from suffix of d
			// append span to l
			span := []int{0, len(d.Suffix())}
			l.spans = append(l.spans, span)
			l.mid = append(l.mid, span)
			tracer().Debugf("prepended suffix span %v", span)
		}
		l.spans = append(l.spans, d.spans...)
		l.mid = append(l.mid, d.spans...)
		if len(d.Prefix()) > 0 && r.HasBoundaries() {
			// make span from prefix of d
			// append span to l
			span := []int{len(unproc) - len(d.Prefix()), len(d.Prefix())}
			l.spans = append(l.spans, span)
			l.mid = append(l.mid, span)
			tracer().Debugf("appended prefix span %v", span)
		}
	}
	l.UnifyWith(&r.MetricValueBase)
	l.spans = append(l.spans, r.spans...)
	return l
}

func (sm *spanningMetric) Leafs(value cords.MetricValue, getBounds bool) []cords.Leaf {
	v := value.(*spanningMetricValue)
	var leafs []cords.Leaf
	if getBounds {
		leafs = make([]cords.Leaf, 2)
		if len(v.Suffix()) != 0 {
			leafs[0] = spanLeaf(len(v.Suffix()))
		}
		if len(v.Prefix()) != 0 {
			leafs[1] = spanLeaf(len(v.Prefix()))
		}
	} else {
		leafs = make([]cords.Leaf, len(v.mid))
		for i, span := range v.mid {
			leafs[i] = spanLeaf(span[1])
			tracer().Debugf("       create leaf = %v from %d…%d", leafs[i], span[0], span[1])
		}
		tracer().Debugf("metric created leafs = %v", leafs)
	}
	return leafs
}

func (sm *spanningMetric) scan(frag []byte, v *spanningMetricValue) *spanningMetricValue {
	s := sm.scnr(frag)
	if s == nil {
		panic(fmt.Sprintf("spanning metric: scanner factory argument failed on '%s'", frag))
	}
	pos, start := 0, 0
	for s.Scan() {
		tracer().Debugf("SCANNED '%s'", s.Text())
		if pos == 0 {
			// first scanned segment is always a suffix
			// this will be done automatically by MetricValueBase if we do not include
			// this suffic into our measured span
			start = len(s.Bytes()) // measured span starts after this suffix
			// we do not recognize a span here
		} else {
			// if this is the last span, it will be converted to a prefix afterwards
			span := []int{pos, len(s.Bytes())}
			v.spans = append(v.spans, span)
			tracer().Debugf("appended span %v", span)
		}
		pos += len(s.Bytes())
	}
	if err := s.Err(); err != nil {
		tracer().Errorf("spanning metric: scanner returned error: %s", err)
		v.lasterr = err
	}
	if len(v.spans) == 1 {
		// remove span and just signal a suffix and a prefix
		v.spans = nospan
		v.split = start
	} else if len(v.spans) > 1 {
		// remove the last span and let it be a prefix
		v.spans = v.spans[:len(v.spans)-1]
	}
	return v
}

func (v *spanningMetricValue) String() string {
	openL, openR := v.Unprocessed()
	return fmt.Sprintf("value{ length=%d, L='%s', R='%s', |P|=%d  }", v.Len(),
		string(openL), string(openR), len(v.spans))
}

var nospan = [][]int{}

func lastpos(spans [][]int) int {
	if len(spans) == 0 {
		return 0
	}
	l := len(spans) - 1
	return spans[l][0] + spans[l][1]
}

// --- Span leafs ------------------------------------------------------------

type spanLeaf uint64

func (leaf spanLeaf) Weight() uint64 {
	return uint64(leaf)
}

func (leaf spanLeaf) String() string {
	return fmt.Sprintf("[%d]", leaf)
}

func (leaf spanLeaf) Substring(i, j uint64) []byte {
	return []byte("X")
}

func (leaf spanLeaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	return spanLeaf(i), leaf - spanLeaf(i)
}
