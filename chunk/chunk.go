package chunk

import (
	"unicode/utf8"
)

// Bitmap indexes byte-local properties inside a chunk.
//
// Bit i corresponds to byte offset i in chunk-local coordinates.
type Bitmap = uint64

const (
	// MaxBase is the maximum chunk payload length in bytes.
	MaxBase = 64
	// MinBase is the minimum non-root occupancy target used by tree policies.
	MinBase = MaxBase / 2
)

// Chunk stores text and bitmap indexes for fast local coordinate math.
//
// The chunk is immutable by convention: editing operations return a new Chunk.
type Chunk struct {
	chars    Bitmap
	newlines Bitmap
	text     [MaxBase]byte
	n        uint8
}

// ChunkSlice is a lightweight view over a chunk range with shifted bitmaps.
type ChunkSlice struct {
	chars    Bitmap
	newlines Bitmap
	text     []byte
}

// New creates a chunk from UTF-8 text.
//
// Returns an error if the text is not valid UTF-8 or exceeds MaxBase bytes.
func New(text string) (Chunk, error) {
	if !utf8.ValidString(text) {
		return Chunk{}, ErrInvalidUTF8
	}
	if len(text) > MaxBase {
		return Chunk{}, ErrChunkTooLarge
	}
	var c Chunk
	copy(c.text[:], text)
	c.n = uint8(len(text))
	// Byte-local ascii properties.
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			c.newlines |= bit(i)
		}
	}
	// Rune-local boundaries.
	for i := range text {
		c.chars |= bit(i)
	}
	return c, nil
}

// NewBytes creates a chunk from UTF-8 bytes.
//
// Returns an error if the bytes are not valid UTF-8 or exceed MaxBase bytes.
//
// Important for file ingestion: callers should split raw input only at UTF-8
// rune boundaries before calling NewBytes for each chunk. This constructor
// validates UTF-8 and will reject byte slices that start/end in the middle of
// a multi-byte rune.
func NewBytes(text []byte) (Chunk, error) {
	if !utf8.Valid(text) {
		return Chunk{}, ErrInvalidUTF8
	}
	if len(text) > MaxBase {
		return Chunk{}, ErrChunkTooLarge
	}
	var c Chunk
	copy(c.text[:], text)
	c.n = uint8(len(text))
	// Byte-local ascii properties.
	for i, b := range text {
		if b == '\n' {
			c.newlines |= bit(i)
		}
	}
	// Rune-local boundaries.
	for i := 0; i < len(text); {
		c.chars |= bit(i)
		_, n := utf8.DecodeRune(text[i:])
		i += n
	}
	return c, nil
}

// Len returns the text length in bytes.
func (c Chunk) Len() int {
	return int(c.n)
}

// IsEmpty reports whether the chunk has no bytes.
func (c Chunk) IsEmpty() bool {
	return c.n == 0
}

// String returns the chunk text.
func (c Chunk) String() string {
	return string(c.text[:c.n])
}

// Bytes returns a copied byte slice of the chunk text.
func (c Chunk) Bytes() []byte {
	return append([]byte(nil), c.text[:c.n]...)
}

// Chars returns the UTF-8 character-start bitmap.
func (c Chunk) Chars() Bitmap {
	return c.chars
}

// Newlines returns the newline bitmap.
func (c Chunk) Newlines() Bitmap {
	return c.newlines
}

// IsCharBoundary reports whether offset is a UTF-8 boundary inside this chunk.
func (c Chunk) IsCharBoundary(offset int) bool {
	if offset == c.Len() {
		return true
	}
	if offset < 0 || offset > c.Len() {
		return false
	}
	return c.chars&bit(offset) != 0
}

// AsSlice returns a zero-offset view over the full chunk.
func (c Chunk) AsSlice() ChunkSlice {
	return ChunkSlice{
		chars:    c.chars,
		newlines: c.newlines,
		text:     c.text[:c.n],
	}
}

// Slice returns a view for [start,end) in chunk-local byte offsets.
func (c Chunk) Slice(start, end int) (ChunkSlice, error) {
	if start < 0 || end < start || end > c.Len() {
		return ChunkSlice{}, ErrIndexOutOfBounds
	}
	if !c.IsCharBoundary(start) || !c.IsCharBoundary(end) {
		return ChunkSlice{}, ErrNotCharBoundary
	}
	m := rangeMask(start, end)
	return ChunkSlice{
		chars:    (c.chars & m) >> uint(start),
		newlines: (c.newlines & m) >> uint(start),
		text:     c.text[start:end],
	}, nil
}

// SplitAt splits a chunk into left/right views at byte offset mid.
func (c Chunk) SplitAt(mid int) (ChunkSlice, ChunkSlice, error) {
	left, err := c.Slice(0, mid)
	if err != nil {
		return ChunkSlice{}, ChunkSlice{}, err
	}
	right, err := c.Slice(mid, c.Len())
	if err != nil {
		return ChunkSlice{}, ChunkSlice{}, err
	}
	return left, right, nil
}

// Append returns a new chunk with slice appended.
//
// The boolean is false if the append would exceed MaxBase; in that case, the
// original chunk is returned unchanged.
func (c Chunk) Append(slice ChunkSlice) (Chunk, bool) {
	if slice.IsEmpty() {
		return c, true
	}
	base := c.Len()
	total := base + slice.Len()
	if total > MaxBase {
		return c, false
	}
	out := c
	shift := uint(base)
	out.chars |= slice.chars << shift
	out.newlines |= slice.newlines << shift
	copy(out.text[base:total], slice.text)
	out.n = uint8(total)
	return out, true
}

// Len returns the slice length in bytes.
func (s ChunkSlice) Len() int {
	return len(s.text)
}

// IsEmpty reports whether the slice has no bytes.
func (s ChunkSlice) IsEmpty() bool {
	return len(s.text) == 0
}

// String returns the slice text.
func (s ChunkSlice) String() string {
	return string(s.text)
}

// Bytes returns a copied byte slice of the slice text.
func (s ChunkSlice) Bytes() []byte {
	return append([]byte(nil), s.text...)
}

// IsCharBoundary reports whether offset is a UTF-8 boundary inside this slice.
func (s ChunkSlice) IsCharBoundary(offset int) bool {
	if offset == s.Len() {
		return true
	}
	if offset < 0 || offset > s.Len() {
		return false
	}
	return s.chars&bit(offset) != 0
}

// Slice returns a sub-view [start,end) in slice-local byte offsets.
func (s ChunkSlice) Slice(start, end int) (ChunkSlice, error) {
	if start < 0 || end < start || end > s.Len() {
		return ChunkSlice{}, ErrIndexOutOfBounds
	}
	if !s.IsCharBoundary(start) || !s.IsCharBoundary(end) {
		return ChunkSlice{}, ErrNotCharBoundary
	}
	m := rangeMask(start, end)
	return ChunkSlice{
		chars:    (s.chars & m) >> uint(start),
		newlines: (s.newlines & m) >> uint(start),
		text:     s.text[start:end],
	}, nil
}

// SplitAt splits a slice into left/right views at byte offset mid.
func (s ChunkSlice) SplitAt(mid int) (ChunkSlice, ChunkSlice, error) {
	left, err := s.Slice(0, mid)
	if err != nil {
		return ChunkSlice{}, ChunkSlice{}, err
	}
	right, err := s.Slice(mid, s.Len())
	if err != nil {
		return ChunkSlice{}, ChunkSlice{}, err
	}
	return left, right, nil
}

// --- Bitmap helpers --------------------------------------------------------

func bit(offset int) Bitmap {
	if offset < 0 || offset >= MaxBase {
		return 0
	}
	return Bitmap(1) << uint(offset)
}

func prefixMask(offset int) Bitmap {
	switch {
	case offset <= 0:
		return 0
	case offset >= MaxBase:
		return ^Bitmap(0)
	default:
		return (Bitmap(1) << uint(offset)) - 1
	}
}

func rangeMask(start, end int) Bitmap {
	return prefixMask(end) &^ prefixMask(start)
}
