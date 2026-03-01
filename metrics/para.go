package metrics

import (
	"math/bits"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
)

// acceptBreak is a predicate function that accepts or rejects a line break.
// We will construct them as context-dependent closures.
type acceptBreak func(byteRange) bool

// BreakPos represents a line break position within a text.
// A `length` of 0 means a non-existent line break.
type BreakPos struct {
	AtByte uint64 // byte offset of the line break
	Length uint16 // number of bytes in the line break
}

func noBreak(at uint64) BreakPos { return BreakPos{at, 0} }

// NextParagraphBreak finds the next paragraph separator at or after byte
// position from, using the given delimiter mode.
//
// The returned separator is a break position.
// For ParagraphByLineBreak, a separator is one recognized line break.
// For ParagraphByBlankLines, a separator is one run of >=2 consecutive line
// breaks.
//
// A return value (bool) of false means there is no further paragraph separator.
func NextParagraphBreak(text cordext.CordEx[btree.NO_EXT], from uint64, mode ParagraphDelimiterPolicy) (
	BreakPos, bool, error) {
	//
	total := text.Len()
	if from > total {
		return noBreak(0), false, cordext.ErrIndexOutOfBounds
	}
	if text.IsVoid() || from == total {
		return noBreak(total), false, nil
	}
	mode = normalizeParagraphPolicy(ParagraphPolicy{Delimiters: mode}).Delimiters
	switch mode {
	case ParagraphByLineBreak:
		br, ok, err := firstLineBreakFrom(text, from)
		if err != nil {
			return noBreak(0), false, err
		}
		if !ok {
			return noBreak(total), false, nil
		}
		//return br.from, br.to, true, nil
		return BreakPos{br.from, uint16(br.to - br.from)}, true, nil
	case ParagraphByBlankLines:
		start, end, found, err := firstBlankLineSeparatorFrom(text, from)
		return BreakPos{start, uint16(end - start)}, found, err
	default:
		return noBreak(0), false, cordext.ErrIllegalArguments
	}
}

// firstLineBreakFrom returns the first line-break range at or after `from`.
//
// It returns (range, true, nil) when a break is found, or (_, false, nil) when
// no break exists at/after `from`. Any iteration error is propagated.
func firstLineBreakFrom(text plainCordType, from uint64) (byteRange, bool, error) {
	var out byteRange
	found := false
	// Build a sink that captures the first emitted break and aborts iteration
	// immediately by returning true.
	err := forEachLineBreak(text, from, func(br byteRange) bool {
		out = br
		found = true
		return true
	})
	return out, found, err
}

// firstBlankLineSeparatorFrom finds the first paragraph separator made of blank
// lines at or after `from`.
//
// A blank-line separator is represented as one contiguous run of at least two
// consecutive line breaks. The returned range [sepFrom, sepTo) spans the whole
// run (for example "\n\n" or "\n\n\n"). If no such run exists, found=false and
// both positions are set to text.Len().
func firstBlankLineSeparatorFrom(text plainCordType, from uint64) (
	sepFrom, sepTo uint64, found bool, err error) {
	//
	var runStart, runEnd uint64
	runLen := 0
	// The sink is a tiny state machine over successive line breaks:
	// it grows the current contiguous run, and stops at the first run whose
	// length reaches 2+ once a gap indicates the run has ended.
	err = forEachLineBreak(text, from, func(br byteRange) bool {
		if runLen == 0 {
			runStart, runEnd, runLen = br.from, br.to, 1
			return false
		}
		if br.from == runEnd {
			runEnd = br.to
			runLen++
			return false
		}
		if runLen >= 2 {
			found = true
			return true
		}
		runStart, runEnd, runLen = br.from, br.to, 1
		return false
	})
	if err != nil {
		return 0, 0, false, err
	}
	if found || runLen >= 2 {
		return runStart, runEnd, true, nil
	}
	// if runLen >= 2 {
	// 	return runStart, runEnd, true, nil
	// }
	return text.Len(), text.Len(), false, nil
}

// locateChunkFrom maps an absolute byte offset `from` to chunk-local coordinates.
//
// It returns:
//   - startItem: item index of the chunk where scanning should start
//   - chunkStart:absolute byte position of that chunk in the full text
//   - localFrom:byte offset inside that chunk where scanning should start
//
// Boundary behavior:
//   - from > text.Len()  => ErrIndexOutOfBounds
//   - empty text         => (0, text.Len(), 0, nil)
//   - from == text.Len() => sentinel "no more bytes": (0, text.Len(), 0, nil)
//   - from at chunk end  => advances to next chunk with localFrom=0
func locateChunkFrom(text plainCordType, from uint64) (
	startItem int64, chunkStart uint64, localFrom int, err error) {
	// TODO I am sure I have similar functions elsewhere; consolidate them
	// in package [cords.cordext].
	//
	total := text.Len()
	p := pipeFor(text, from <= total)
	if p.err == ErrVoidText || from == total {
		return 0, total, 0, nil
	}
	// if p.err != nil {
	// 	return 0, 0, 0, p.err
	// }
	// if from > total {
	// 	return 0, 0, 0, cordext.ErrIndexOutOfBounds
	// }
	// tree := text.Tree()
	// if tree == nil || tree.IsEmpty() || from == text.Len() {
	// 	return 0, text.Len(), 0, nil
	// }
	tree := text.Tree()
	// I am unable to convince the compiler about the correctness of the types
	// when doing a pipeline call, therefore stick to explicit call.
	var cursor *btree.Cursor[chunk.Chunk, chunk.Summary, btree.NO_EXT, uint64]
	if p.err == nil {
		cursor, p.err = btree.NewCursor(tree, chunk.ByteDimension{})
	}
	// if err != nil {
	// 	return 0, 0, 0, err
	// }
	idx, acc := pipeCall1to2(p, cursor.Seek, from)
	// idx, acc, err := cursor.Seek(from)
	// if err != nil {
	// 	return 0, 0, 0, err
	// }

	if p.err == nil {
		assert(idx < tree.Len(), "metrics: internal error, cursor seek inconsistent")
	}

	// if p.err != nil {
	// 	return 0, 0, 0, p.err
	// }
	first := pipeCall1(p, tree.At, idx)
	// first, err := tree.At(idx)
	// if err != nil {
	// 	return 0, 0, 0, err
	// }

	if p.err != nil {
		return 0, 0, 0, p.err
	} else if from == 0 {
		return idx, 0, 0, nil
	}
	start := acc - uint64(first.Len())
	local := int(from - start)
	if local == first.Len() {
		idx++
		if idx >= tree.Len() {
			return idx, total, 0, nil
		}
		return idx, from, 0, nil
	}
	return idx, start, local, nil
}

// forEachLineBreak emits all line-break byte ranges at or after
// absolute byte position `from`.
//
// It is optimized for scanning large cords:
//   - locateChunkFrom maps `from` to a start chunk plus local byte offset.
//   - Iteration then walks chunks from that point onward.
//   - For each chunk, the newline bitmap (chunk.Newlines()) is used to visit only
//     candidate newline positions, instead of scanning all bytes.
//
// Emitted ranges are:
// - LF:   [pos, pos+1)
// - CRLF: [pos-1, pos+1) when '\r' directly precedes '\n'
//
// CRLF pairs split across chunk boundaries are handled via pendingCR/pendingCRPos.
// The sink returns true to accept/stop iteration, or false to continue.
func forEachLineBreak(text plainCordType, from uint64, sink acceptBreak) error {
	tree := text.Tree()
	assert(tree != nil && !tree.IsEmpty(), "metrics: internal error, tree is nil or empty")
	// if tree == nil || tree.IsEmpty() {
	// 	return nil
	// }
	startItem, chunkStart, localFrom, err := locateChunkFrom(text, from)
	assert(startItem < tree.Len(), "metrics: internal error, cursor seek inconsistent")
	//if err != nil || startItem >= tree.Len() {
	if err != nil {
		return err
	}
	buf := make([]byte, 0, chunk.MaxBase)
	firstChunk := true
	var pendingCR bool
	var pendingCRPos uint64
	// We already found the starting chunk and the local offset within it
	// Now we iterate over all chunks starting from this one
	for _, chnk := range tree.ItemRange(startItem, tree.Len()) {
		bytes := chnk.Bytes(buf)
		startOff := 0
		if firstChunk {
			startOff = localFrom
			firstChunk = false
		}
		startOff = max(0, startOff)
		// if startOff < 0 {
		// 	startOff = 0
		// }
		startOff = min(startOff, len(bytes))
		// if startOff > len(bytes) {
		// 	startOff = len(bytes)
		// }

		consumedFirstLF := false
		if pendingCR { // if the last byte of the previous chunk was a CR
			if startOff == 0 && len(bytes) > 0 && bytes[0] == '\n' {
				br := byteRange{from: pendingCRPos, to: chunkStart + 1}
				if br.from >= from {
					if sink(br) {
						return nil
					}
				}
				consumedFirstLF = true
			}
			pendingCR = false
		}

		// We skip chunks where the summary tells us there are no line breaks in it
		if chnk.Summary().Lines > 0 {
			mask := uint64(chnk.Newlines())
			mask = clearBitsBelow(mask, startOff)
			if consumedFirstLF {
				mask &^= 1
			}
			for mask != 0 {
				i := bits.TrailingZeros64(mask)
				nlPos := chunkStart + uint64(i)
				br := byteRange{from: nlPos, to: nlPos + 1}
				if i > 0 && bytes[i-1] == '\r' {
					br.from = nlPos - 1
				}
				if br.from >= from {
					if sink(br) {
						return nil
					}
				}
				mask &= mask - 1
			}
		}

		// Carry only if this chunk contributes bytes at/after `from`.
		if startOff < len(bytes) && len(bytes) > 0 && bytes[len(bytes)-1] == '\r' {
			pendingCR = true
			pendingCRPos = chunkStart + uint64(len(bytes)-1)
		}
		chunkStart += uint64(len(bytes))
	}
	return nil
}

// clearBitsBelow is a small helper to clear all bits below a given bit position
func clearBitsBelow(mask uint64, n int) uint64 {
	switch {
	case n <= 0:
		return mask
	case n >= 64:
		return 0
	default:
		return mask &^ ((uint64(1) << uint(n)) - 1)
	}
}
