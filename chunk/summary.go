package chunk

import "math/bits"

// Summary aggregates chunk-level text metrics for tree routing.
//
// Tree-level code uses this summary to navigate and aggregate, while chunk
// code keeps ownership of local byte/rune boundary logic.
type Summary struct {
	Bytes uint64
	Chars uint64
	Lines uint64
}

// Summary returns aggregate metrics for this chunk.
func (c Chunk) Summary() Summary {
	return summarize(c.Len(), c.chars, c.newlines)
}

// Summary returns aggregate metrics for this chunk view.
func (s ChunkSlice) Summary() Summary {
	return summarize(s.Len(), s.chars, s.newlines)
}

func summarize(n int, chars Bitmap, newlines Bitmap) Summary {
	mask := prefixMask(n)
	return Summary{
		Bytes: uint64(n),
		Chars: uint64(bits.OnesCount64(uint64(chars & mask))),
		Lines: uint64(bits.OnesCount64(uint64(newlines & mask))),
	}
}

// Monoid aggregates chunk summaries for B+ sum-tree internal nodes.
type Monoid struct{}

// Zero returns the neutral summary value.
func (Monoid) Zero() Summary { return Summary{} }

// Add combines two summaries.
func (Monoid) Add(left, right Summary) Summary {
	return Summary{
		Bytes: left.Bytes + right.Bytes,
		Chars: left.Chars + right.Chars,
		Lines: left.Lines + right.Lines,
	}
}

// ByteDimension seeks by byte count in tree summaries.
type ByteDimension struct{}

// Zero returns the byte origin.
func (ByteDimension) Zero() uint64 { return 0 }

// Add accumulates bytes from summary into the dimension accumulator.
func (ByteDimension) Add(acc uint64, summary Summary) uint64 {
	return acc + summary.Bytes
}

// Compare compares accumulated value to a seek target.
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

// CharDimension seeks by Unicode scalar value count.
type CharDimension struct{}

// Zero returns the character origin.
func (CharDimension) Zero() uint64 { return 0 }

// Add accumulates character counts from summary.
func (CharDimension) Add(acc uint64, summary Summary) uint64 {
	return acc + summary.Chars
}

// Compare compares accumulated value to a seek target.
func (CharDimension) Compare(acc uint64, target uint64) int {
	switch {
	case acc < target:
		return -1
	case acc > target:
		return 1
	default:
		return 0
	}
}

// LineDimension seeks by newline count.
type LineDimension struct{}

// Zero returns the line origin.
func (LineDimension) Zero() uint64 { return 0 }

// Add accumulates newline counts from summary.
func (LineDimension) Add(acc uint64, summary Summary) uint64 {
	return acc + summary.Lines
}

// Compare compares accumulated value to a seek target.
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
