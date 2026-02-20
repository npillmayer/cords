package cords

import "github.com/npillmayer/cords/chunk"

// TextSegment is a cords-level read-only view of one text chunk and its summary.
//
// It is intended as a stable API surface for extension and analytics code so
// callers do not need to depend on btree internals.
type TextSegment struct {
	chunk   chunk.Chunk
	summary chunk.Summary
}

func newTextSegment(c chunk.Chunk) TextSegment {
	return newTextSegmentWithSummary(c, c.Summary())
}

func newTextSegmentWithSummary(c chunk.Chunk, s chunk.Summary) TextSegment {
	return TextSegment{
		chunk:   c,
		summary: s,
	}
}

// Chunk returns the underlying chunk value.
func (s TextSegment) Chunk() chunk.Chunk {
	return s.chunk
}

// Summary returns the segment summary (bytes/chars/lines).
func (s TextSegment) Summary() chunk.Summary {
	return s.summary
}

// ByteLen returns the number of bytes in this segment.
func (s TextSegment) ByteLen() uint64 {
	return s.summary.Bytes
}

// CharCount returns the number of UTF-8 runes in this segment.
func (s TextSegment) CharCount() uint64 {
	return s.summary.Chars
}

// LineCount returns the number of newline characters in this segment.
func (s TextSegment) LineCount() uint64 {
	return s.summary.Lines
}

// Len returns the number of bytes in this segment.
func (s TextSegment) Len() int {
	return s.chunk.Len()
}

// IsEmpty reports whether the segment has no bytes.
func (s TextSegment) IsEmpty() bool {
	return s.chunk.IsEmpty()
}

// String returns the segment text.
func (s TextSegment) String() string {
	return s.chunk.String()
}

// Bytes returns a copied byte slice of the segment text.
func (s TextSegment) Bytes() []byte {
	return s.chunk.Bytes()
}

// Chars returns the UTF-8 character-start bitmap for this segment.
func (s TextSegment) Chars() chunk.Bitmap {
	return s.chunk.Chars()
}

// Newlines returns the newline bitmap for this segment.
func (s TextSegment) Newlines() chunk.Bitmap {
	return s.chunk.Newlines()
}

// IsCharBoundary reports whether offset is a UTF-8 boundary inside this segment.
func (s TextSegment) IsCharBoundary(offset int) bool {
	return s.chunk.IsCharBoundary(offset)
}
