package btree

import "bytes"

// TextSummary is a default summary type for text chunks.
//
// It is intentionally small and additive so it can serve as a base for
// dimensioned cursor operations.
type TextSummary struct {
	Bytes uint64
	Lines uint64
}

// TextChunk is a rope leaf item that is summarized at the type level.
type TextChunk []byte

// FromString creates a text chunk from a Go string.
func FromString(s string) TextChunk {
	return TextChunk([]byte(s))
}

// Summary returns bytes/lines for this chunk.
func (chunk TextChunk) Summary() TextSummary {
	return TextSummary{
		Bytes: uint64(len(chunk)),
		Lines: uint64(bytes.Count(chunk, []byte{'\n'})),
	}
}

// TextMonoid aggregates TextSummary values.
type TextMonoid struct{}

// Zero returns the neutral summary.
func (TextMonoid) Zero() TextSummary {
	return TextSummary{}
}

// Add combines two summaries.
func (TextMonoid) Add(left, right TextSummary) TextSummary {
	return TextSummary{
		Bytes: left.Bytes + right.Bytes,
		Lines: left.Lines + right.Lines,
	}
}

// ByteDimension seeks/accumulates by byte count.
type ByteDimension struct{}

// Zero returns 0 bytes.
func (ByteDimension) Zero() uint64 { return 0 }

// Add adds bytes from summary into accumulator.
func (ByteDimension) Add(acc uint64, summary TextSummary) uint64 {
	return acc + summary.Bytes
}

// Compare compares dimension progress to target.
func (ByteDimension) Compare(acc uint64, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}

// LineDimension seeks/accumulates by newline count.
type LineDimension struct{}

// Zero returns 0 lines.
func (LineDimension) Zero() uint64 { return 0 }

// Add adds lines from summary into accumulator.
func (LineDimension) Add(acc uint64, summary TextSummary) uint64 {
	return acc + summary.Lines
}

// Compare compares dimension progress to target.
func (LineDimension) Compare(acc uint64, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}
