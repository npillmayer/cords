package cords

import "io"

// Reader returns a reader for the bytes of cord.
func (cord Cord) Reader() io.Reader {
	return &cordReader{cord: cord}
}

type cordReader struct {
	cord   Cord
	cursor uint64
}

func (cr *cordReader) Read(p []byte) (n int, err error) {
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
