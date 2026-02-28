package cordext

import (
	"io"
	"iter"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
)

// RangeChunk returns an iterator over all chunks in logical order.
func (cord CordEx[E]) RangeChunk() iter.Seq[chunk.Chunk] {
	return func(yield func(chunk.Chunk) bool) {
		for seg := range cord.RangeTextSegment() {
			if !yield(seg.Chunk()) {
				return
			}
		}
	}
}

// RangeTextSegment returns an iterator over all text segments in logical order.
func (cord CordEx[E]) RangeTextSegment() iter.Seq[TextSegment] {
	return func(yield func(TextSegment) bool) {
		if cord.tree == nil {
			return
		}
		cord.tree.ForEachItem(func(c chunk.Chunk) bool {
			return yield(newTextSegment(c))
		})
	}
}

// RangeChunkBounded returns a bounded iterator over text chunks in logical order.
func (cord CordEx[E]) ChunkRangeBounded(from, to int64) iter.Seq2[int64, chunk.Chunk] {
	if cord.tree == nil {
		return func(yield func(int64, chunk.Chunk) bool) {}
	}
	from, to = max(0, from), min(to, cord.tree.Len())
	if to <= from {
		return func(yield func(int64, chunk.Chunk) bool) {}
	}
	tracer().Debugf("cordext: bounded chunk range %d..%d", from, to)
	return cord.tree.ItemRange(from, to)
}

// TextSegmentRangeBounded returns a bounded iterator over text-segments in logical
// order.
func (cord CordEx[E]) TextSegmentRangeBounded(from, to int64) iter.Seq[TextSegment] {
	if cord.tree == nil {
		return func(yield func(TextSegment) bool) {}
	}
	from, to = max(0, from), min(to, cord.tree.Len())
	if to <= from {
		return func(yield func(TextSegment) bool) {}
	}
	return func(yield func(TextSegment) bool) {
		for _, chunk := range cord.ChunkRangeBounded(from, to) {
			seg := newTextSegment(chunk)
			if !yield(seg) {
				return
			}
		}
	}
}

// EachChunk visits all chunks in logical order.
func (cord CordEx[E]) EachChunk(f func(chunk.Chunk, uint64) error) error {
	return cord.EachTextSegment(func(seg TextSegment, pos uint64) error {
		return f(seg.Chunk(), pos)
	})
}

// EachTextSegment visits all text segments in logical order.
func (cord CordEx[E]) EachTextSegment(f func(TextSegment, uint64) error) error {
	if cord.tree == nil {
		return nil
	}
	var err error
	var pos uint64
	cord.tree.ForEachItem(func(c chunk.Chunk) bool {
		if err != nil {
			return false
		}
		err = f(newTextSegment(c), pos)
		pos += c.Summary().Bytes
		return err == nil
	})
	return err
}

// --- Iterating over Bytes --------------------------------------------------

func byteCursor[E any](cord CordEx[E]) *btree.Cursor[chunk.Chunk, chunk.Summary, E, uint64] {
	assert(cord.tree != nil, "cordext: no cursor for nil-cord")
	tree := cord.tree
	cursor, _ := btree.NewCursor(tree, chunk.ByteDimension{})
	return cursor
}

var noIterator = func(yield func(int, byte) bool) {}

func (cord CordEx[E]) ByteRangeBounded(from, to uint64) iter.Seq2[int, byte] {
	if cord.tree == nil {
		return noIterator
	}
	from, to = max(0, from), min(to, cord.Len())
	if to <= from {
		return noIterator
	}
	cursor := byteCursor(cord)
	startItemInx, acc, err := cursor.Seek(from)
	if err != nil {
		return noIterator
	}
	tracer().Debugf("cordext: cursor acc = %d at item %d", acc, startItemInx)
	chnk, err := cord.tree.At(startItemInx)
	if err != nil {
		return noIterator
	}
	endItemInx, acc, err := cursor.Seek(to)
	if err != nil {
		return noIterator
	}
	tracer().Debugf("cordext: cursor acc = %d at item %d", acc, endItemInx)
	left := int(acc - uint64(chnk.Len()) + from)
	right := int(acc - uint64(chnk.Len()) + to)
	inx := int(acc - uint64(chnk.Len()))
	buf := make([]byte, 0, chunk.MaxBase)
	return func(yield func(int, byte) bool) {
		for j, chnk := range cord.ChunkRangeBounded(startItemInx, endItemInx+1) {
			tracer().Debugf("cordext: chunk %d at %d", j, inx)
			buf = chnk.Bytes(buf)
			for i, b := range buf {
				if inx+i < left {
					continue
				}
				if inx+i >= right {
					return
				}
				if !yield(inx+i, b) {
					return
				}
			}
			inx += chnk.Len()
		}
	}
}

// Readers are implemented in terms of iterators. Package [iter] supports wrapping
// iterators as pull-based readers.
//
// https://go.dev/blog/range-functions#pull-iterators
//
// ---Bounded Reader on cord -------------------------------------------------

func (cord CordEx[E]) BoundedReader(from, to uint64) io.Reader {
	if cord.tree == nil {
		return nil
	}
	from, to = max(0, from), min(to, cord.Len())
	if to <= from {
		return nil
	}
	return &boundedReader[E]{
		iterator: cord.ByteRangeBounded(from, to),
	}
}

type boundedReader[E any] struct {
	iterator iter.Seq2[int, byte]
	next     func() (int, byte, bool)
	stop     func()
}

func (r *boundedReader[E]) Read(p []byte) (n int, err error) {
	if r.next == nil {
		r.next, r.stop = iter.Pull2(r.iterator)
	}
	for n = range len(p) {
		_, b, ok := r.next()
		if !ok {
			err = io.EOF
			defer r.stop()
			return
		}
		p[n] = b
	}
	return
}

// --- Reader for whole cord -------------------------------------------------

type zeroReader struct{}

func (z zeroReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

// Reader returns a sequential reader over cord bytes.
//
// The reader is non-mutating and reads from byte offset 0 to Len()-1.
func (cord CordEx[E]) Reader() io.Reader {
	if cord.tree == nil {
		return zeroReader{}
	}
	return &cordReaderEx{
		chunkIterator: cord.RangeChunk(),
	}
}

type cordReaderEx struct {
	currentChunk  []byte
	chnkInx       int                        // index into current chunk
	chunkIterator iter.Seq[chunk.Chunk]      // we will iterate over all chungs of a cord
	pullChunk     func() (chunk.Chunk, bool) // get the next chunk
	stop          func()                     // stop the iterator
}

func (cr *cordReaderEx) Read(p []byte) (n int, err error) {
	if cr.pullChunk == nil {
		cr.pullChunk, cr.stop = iter.Pull(cr.chunkIterator)
	}
	for n = range len(p) {
		if cr.chnkInx >= len(cr.currentChunk) {
			chnk, ok := cr.pullChunk()
			if !ok {
				tracer().Debugf("cordext: no more chunks, n=%d", n)
				err = io.EOF
				cr.stop()
				return
			}
			tracer().Debugf("cordext: pulled chunk = %s", chnk.String())
			if cr.currentChunk == nil {
				cr.currentChunk = make([]byte, 0, chunk.MaxBase)
			}
			cr.currentChunk = chnk.Bytes(cr.currentChunk)
			cr.chnkInx = 0
		}
		b := cr.currentChunk[cr.chnkInx]
		tracer().Debugf("cordext: read byte = %d ('%c')", b, b)
		p[n] = b
		cr.chnkInx++
	}
	// From the documentation of io.Reader:
	// If len(p) == 0, Read should always return n == 0. It may return a non-nil error if
	// some error condition is known, such as EOF.
	// Implementations of Read are discouraged from returning a zero byte count with a
	// nil error, except when len(p) == 0. Callers should treat a return of 0 and nil as
	// indicating that nothing happened; in particular it does not indicate EOF.
	if len(p) > 0 {
		n += 1 // has been len(p)-1 after range loop [ 0..len(p) )
	}
	return
}
