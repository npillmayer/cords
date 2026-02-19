package cords

import (
	"unicode/utf8"
)

// CharCursor navigates a cord by UTF-8 rune positions.
//
// The cursor is bound to one cord snapshot. Movement is in rune steps, while
// internal addressing uses byte offsets for efficient tree routing.
type CharCursor struct {
	cord    Cord
	pos     Pos
	byteOff uint64
}

// NewCharCursor creates a rune-aware cursor at the start of cord.
func (cord Cord) NewCharCursor() (*CharCursor, error) {
	start := cord.PosStart()
	return &CharCursor{
		cord:    cord,
		pos:     start,
		byteOff: start.bytepos,
	}, nil
}

// Pos returns the current immutable cursor position.
func (cc *CharCursor) Pos() Pos {
	if cc == nil {
		return Pos{}
	}
	return cc.pos
}

// ByteOffset returns the current cursor byte offset.
func (cc *CharCursor) ByteOffset() uint64 {
	if cc == nil {
		return 0
	}
	return cc.byteOff
}

// SeekPos moves the cursor to p after validating p against the cursor's cord.
func (cc *CharCursor) SeekPos(p Pos) error {
	if cc == nil {
		return ErrIllegalArguments
	}
	if err := cc.cord.validatePos(p); err != nil {
		return err
	}
	cc.pos = p
	cc.byteOff = p.bytepos
	return nil
}

// SeekRunes moves the cursor to absolute rune offset n.
func (cc *CharCursor) SeekRunes(n uint64) error {
	if cc == nil {
		return ErrIllegalArguments
	}
	p, err := cc.cord.posFromRunes(n)
	if err != nil {
		return err
	}
	cc.pos = p
	cc.byteOff = p.bytepos
	return nil
}

// Next returns the rune at the current cursor position and advances by one rune.
//
// If the cursor is at end-of-cord, ok is false.
func (cc *CharCursor) Next() (r rune, ok bool) {
	if cc == nil {
		return 0, false
	}
	if cc.byteOff >= cc.cord.Len() {
		return 0, false
	}

	item, local, err := cc.cord.Index(cc.byteOff)
	if err != nil {
		return 0, false
	}
	b := item.Bytes()
	off := int(local)
	if off < 0 || off >= len(b) {
		return 0, false
	}
	r, n := utf8.DecodeRune(b[off:])
	if r == utf8.RuneError && n == 1 {
		return 0, false
	}
	cc.byteOff += uint64(n)
	cc.pos = Pos{runes: cc.pos.runes + 1, bytepos: cc.byteOff}
	return r, true
}

// Prev returns the rune before the current cursor position and moves back by one rune.
//
// If the cursor is at start-of-cord, ok is false.
func (cc *CharCursor) Prev() (r rune, ok bool) {
	if cc == nil {
		return 0, false
	}
	if cc.byteOff == 0 {
		return 0, false
	}

	probe := cc.byteOff - 1
	item, local, err := cc.cord.Index(probe)
	if err != nil {
		return 0, false
	}
	b := item.Bytes()
	off := int(local)
	if off < 0 || off >= len(b) {
		return 0, false
	}
	for off > 0 && !utf8.RuneStart(b[off]) {
		off--
	}
	r, n := utf8.DecodeRune(b[off:])
	if r == utf8.RuneError && n == 1 {
		return 0, false
	}

	chunkStart := probe - local
	newByte := chunkStart + uint64(off)
	if newByte > cc.byteOff {
		return 0, false
	}
	cc.byteOff = newByte
	if cc.pos.runes > 0 {
		cc.pos = Pos{runes: cc.pos.runes - 1, bytepos: cc.byteOff}
	} else {
		// Defensive fallback; should not happen when byteOff > 0 in valid UTF-8.
		cc.pos = Pos{bytepos: cc.byteOff}
	}
	return r, true
}
