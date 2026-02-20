package cords

import (
	"errors"
	"fmt"
	"math/bits"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
)

// Pos is an immutable rune-aware position coordinate.
//
// A Pos carries both a rune offset and a byte offset. Both values always refer
// to the same logical boundary in a specific cord snapshot.
type Pos struct {
	runes   uint64
	bytepos uint64
}

// PosStart returns the zero position of a cord.
func (cord Cord) PosStart() Pos {
	return Pos{}
}

// PosEnd returns the end position of a cord.
func (cord Cord) PosEnd() Pos {
	tree, err := treeFromCord(cord)
	assert(err == nil, "cord.PosEnd: cannot materialize tree")
	if tree == nil {
		return Pos{}
	}
	s := tree.Summary()
	return Pos{runes: s.Chars, bytepos: s.Bytes}
}

// PosFromByte creates a rune-aware position from a byte offset.
//
// The byte offset must point to a UTF-8 rune boundary.
func (cord Cord) PosFromByte(b uint64) (Pos, error) {
	tree, err := treeFromCord(cord)
	if err != nil {
		return Pos{}, err
	}
	total := tree.Summary()
	if b > total.Bytes {
		return Pos{}, ErrIndexOutOfBounds
	}
	if b == 0 {
		return Pos{}, nil
	}
	if b == total.Bytes {
		return Pos{runes: total.Chars, bytepos: b}, nil
	}

	byteCur, err := btree.NewCursor[chunk.Chunk, chunk.Summary, btree.NO_EXT, uint64](tree, chunk.ByteDimension{})
	if err != nil {
		return Pos{}, err
	}
	itemIndex, acc, err := byteCur.Seek(b)
	if err != nil {
		return Pos{}, err
	}
	if itemIndex < 0 || itemIndex >= tree.Len() {
		return Pos{}, ErrIndexOutOfBounds
	}

	item, err := tree.At(itemIndex)
	if err != nil {
		return Pos{}, err
	}
	itemBytes := item.Summary().Bytes
	beforeBytes := acc - itemBytes
	localByte := int(b - beforeBytes)
	localRunes, err := chunkRunesBeforeByte(item, localByte)
	if err != nil {
		return Pos{}, err
	}

	prefix, err := prefixSummaryBeforeItem(tree, itemIndex)
	if err != nil {
		return Pos{}, err
	}
	return Pos{runes: prefix.Chars + localRunes, bytepos: b}, nil
}

// ByteOffset returns the byte offset for a rune-aware position.
//
// The position is validated against the receiving cord.
func (cord Cord) ByteOffset(p Pos) (uint64, error) {
	if err := cord.validatePos(p); err != nil {
		return 0, err
	}
	return p.bytepos, nil
}

// posFromRunes converts a rune offset to a Pos.
//
// This helper stays internal until the rune-position API surface is finalized.
func (cord Cord) posFromRunes(r uint64) (Pos, error) {
	tree, err := treeFromCord(cord)
	if err != nil {
		return Pos{}, err
	}
	total := tree.Summary()
	if r > total.Chars {
		return Pos{}, ErrIndexOutOfBounds
	}
	if r == 0 {
		return Pos{}, nil
	}
	if r == total.Chars {
		return Pos{runes: r, bytepos: total.Bytes}, nil
	}

	charCur, err := btree.NewCursor[chunk.Chunk, chunk.Summary, btree.NO_EXT, uint64](tree, chunk.CharDimension{})
	if err != nil {
		return Pos{}, err
	}
	itemIndex, acc, err := charCur.Seek(r)
	if err != nil {
		return Pos{}, err
	}
	if itemIndex < 0 || itemIndex >= tree.Len() {
		return Pos{}, ErrIndexOutOfBounds
	}

	item, err := tree.At(itemIndex)
	if err != nil {
		return Pos{}, err
	}
	itemChars := item.Summary().Chars
	beforeChars := acc - itemChars
	localRunes := r - beforeChars
	localByte, err := chunkByteForRuneCount(item, localRunes)
	if err != nil {
		return Pos{}, err
	}

	prefix, err := prefixSummaryBeforeItem(tree, itemIndex)
	if err != nil {
		return Pos{}, err
	}
	return Pos{runes: r, bytepos: prefix.Bytes + uint64(localByte)}, nil
}

// validatePos verifies that a Pos is consistent for the receiving cord.
func (cord Cord) validatePos(p Pos) error {
	if p.bytepos > cord.Len() {
		return ErrIndexOutOfBounds
	}
	resolved, err := cord.PosFromByte(p.bytepos)
	if err != nil {
		if errors.Is(err, ErrIndexOutOfBounds) {
			return err
		}
		return fmt.Errorf("%w: cannot resolve byte offset", ErrIllegalPosition)
	}
	if resolved.runes != p.runes {
		return ErrIllegalPosition
	}
	return nil
}

func prefixSummaryBeforeItem(tree *btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT], itemIndex int) (chunk.Summary, error) {
	return tree.PrefixSummary(itemIndex)
}

func chunkRunesBeforeByte(c chunk.Chunk, localByte int) (uint64, error) {
	if localByte < 0 || localByte > c.Len() {
		return 0, ErrIndexOutOfBounds
	}
	if !c.IsCharBoundary(localByte) {
		return 0, ErrIllegalPosition
	}
	var mask uint64
	switch {
	case localByte <= 0:
		mask = 0
	case localByte >= chunk.MaxBase:
		mask = ^uint64(0)
	default:
		mask = (uint64(1) << uint(localByte)) - 1
	}
	return uint64(bits.OnesCount64(uint64(c.Chars()) & mask)), nil
}

func chunkByteForRuneCount(c chunk.Chunk, runes uint64) (int, error) {
	if runes > c.Summary().Chars {
		return 0, ErrIndexOutOfBounds
	}
	if runes == 0 {
		return 0, nil
	}

	seen := uint64(0)
	for i := 0; i <= c.Len(); i++ {
		if i == c.Len() {
			if seen == runes {
				return i, nil
			}
			break
		}
		if !c.IsCharBoundary(i) {
			continue
		}
		if seen == runes {
			return i, nil
		}
		seen++
	}
	return 0, ErrIllegalPosition
}
