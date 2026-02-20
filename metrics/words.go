package metrics

import (
	"unicode"
	"unicode/utf8"

	"github.com/npillmayer/cords"
)

// Span is a byte-range descriptor inside a cord snapshot.
//
// Pos is the start byte offset, Len is the span length in bytes.
type Span struct {
	Pos uint64
	Len uint64
}

// WordsValue is the result of a word-materialization pass.
type WordsValue struct {
	Spans []Span
}

// WordCount returns the number of recognized words.
func (v WordsValue) WordCount() int {
	return len(v.Spans)
}

// WordsMetric is a materialized word metric.
type WordsMetric struct{}

// Words creates a materialized word metric.
func Words() WordsMetric {
	return WordsMetric{}
}

// Apply scans [i,j) for words and returns word spans plus a materialized cord.
//
// Materialization concatenates all recognized words in logical order and omits
// non-word separators.
func (WordsMetric) Apply(text cords.Cord, i, j uint64) (WordsValue, cords.Cord, error) {
	if text.IsVoid() {
		return WordsValue{}, cords.Cord{}, nil
	}
	if i > text.Len() || j > text.Len() || j < i {
		return WordsValue{}, cords.Cord{}, cords.ErrIndexOutOfBounds
	}
	if i == j {
		return WordsValue{}, cords.Cord{}, nil
	}
	content, err := text.Report(i, j-i)
	if err != nil {
		return WordsValue{}, cords.Cord{}, err
	}
	contentBytes := []byte(content)
	value := WordsValue{
		Spans: findWordSpans(contentBytes, i),
	}
	if len(value.Spans) == 0 {
		return value, cords.Cord{}, nil
	}

	totalBytes := 0
	for _, span := range value.Spans {
		totalBytes += int(span.Len)
	}
	out := make([]byte, 0, totalBytes)
	for _, span := range value.Spans {
		start := int(span.Pos - i)
		end := start + int(span.Len)
		out = append(out, contentBytes[start:end]...)
	}
	return value, cords.FromString(string(out)), nil
}

func findWordSpans(b []byte, base uint64) []Span {
	spans := make([]Span, 0, 8)
	for pos := 0; pos < len(b); {
		r, width := utf8.DecodeRune(b[pos:])
		if r == utf8.RuneError && width == 1 {
			width = 1
		}
		if unicode.IsSpace(r) {
			pos += width
			continue
		}
		start := pos
		pos += width
		for pos < len(b) {
			r, width = utf8.DecodeRune(b[pos:])
			if r == utf8.RuneError && width == 1 {
				width = 1
			}
			if unicode.IsSpace(r) {
				break
			}
			pos += width
		}
		spans = append(spans, Span{
			Pos: base + uint64(start),
			Len: uint64(pos - start),
		})
	}
	return spans
}
