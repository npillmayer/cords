package cords

import (
	"bytes"
	"io"
)

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
	//T().Debugf("l=%d", l)
	if cr.cursor+l > cr.cord.Len() {
		l = cr.cord.Len() - cr.cursor
		if l == 0 {
			return 0, io.EOF
		}
	}
	i := cr.cursor
	buf := bytes.NewBuffer(p[:0])
	substr(&cr.cord.root.cordNode, i, i+l, buf)
	if uint64(buf.Len()) > l {
		panic("cord reader: bytes.Buffer has grown byte array")
	}
	cr.cursor += l
	//T().Debugf("buf=%v, buf.Bytes()=%v", buf, buf.Bytes())
	return int(l), nil
}
