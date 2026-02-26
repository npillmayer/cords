package cordext

import "io"

// Reader returns a sequential reader over cord bytes.
//
// The reader is non-mutating and reads from byte offset 0 to Len().
func (cord CordEx[E]) Reader() io.Reader {
	return &cordReaderEx[E]{cord: cord}
}

type cordReaderEx[E any] struct {
	cord   CordEx[E]
	cursor uint64
}

// Read implements io.Reader.
//
// It copies at most len(p) bytes from the current cursor and advances the cursor
// by the number of bytes returned.
func (cr *cordReaderEx[E]) Read(p []byte) (n int, err error) {
	l := uint64(len(p))
	if cr.cursor+l > cr.cord.Len() {
		l = cr.cord.Len() - cr.cursor
		if l == 0 {
			return 0, io.EOF
		}
	}
	i := cr.cursor
	s, err := cr.cord.Report(i, l)
	if err != nil {
		return 0, err
	}
	n = copy(p, s)
	cr.cursor += uint64(n)
	return n, nil
}
