package metrics

import (
	"errors"
	"strings"
	"testing"

	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestLocateChunkFrom(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.cords")
	defer teardown()
	//
	text := strings.Repeat("x", chunk.MaxBase) + "yz"
	c := cordext.FromStringNoExt(text)

	idx, start, local, err := locateChunkFrom(c, 0)
	if err != nil {
		t.Fatalf("locateChunkFrom(0) failed: %v", err)
	}
	if idx != 0 || start != 0 || local != 0 {
		t.Fatalf("locateChunkFrom(0): got idx=%d start=%d local=%d", idx, start, local)
	}

	idx, start, local, err = locateChunkFrom(c, 10)
	if err != nil {
		t.Fatalf("locateChunkFrom(10) failed: %v", err)
	}
	if idx != 0 || start != 0 || local != 10 {
		t.Fatalf("locateChunkFrom(10): got idx=%d start=%d local=%d", idx, start, local)
	}

	idx, start, local, err = locateChunkFrom(c, chunk.MaxBase)
	if err != nil {
		t.Fatalf("locateChunkFrom(MaxBase) failed: %v", err)
	}
	if idx != 1 || start != chunk.MaxBase || local != 0 {
		t.Fatalf("locateChunkFrom(MaxBase): got idx=%d start=%d local=%d", idx, start, local)
	}

	idx, start, local, err = locateChunkFrom(c, chunk.MaxBase+1)
	if err != nil {
		t.Fatalf("locateChunkFrom(MaxBase+1) failed: %v", err)
	}
	if idx != 1 || start != chunk.MaxBase || local != 1 {
		t.Fatalf("locateChunkFrom(MaxBase+1): got idx=%d start=%d local=%d", idx, start, local)
	}

	idx, start, local, err = locateChunkFrom(c, c.Len())
	if err != nil {
		t.Fatalf("locateChunkFrom(len) failed: %v", err)
	}
	if idx != 0 || start != c.Len() || local != 0 {
		t.Fatalf("locateChunkFrom(len): got idx=%d start=%d local=%d", idx, start, local)
	}

	if _, _, _, err := locateChunkFrom(c, c.Len()+1); !errors.Is(err, ErrIllegalArguments) {
		t.Fatalf("expected ErrIllegalArguments for from=len+1, got %v", err)
	}
}

func TestNextParagraphBreakLineMode(t *testing.T) {
	c := cordext.FromStringNoExt("A\nB\nC")
	br, found, err := NextParagraphBreak(c, 0, ParagraphByLineBreak)
	if err != nil {
		t.Fatalf("NextParagraphBreak(0) failed: %v", err)
	}
	if !found || br.AtByte != 1 || br.Length != 1 {
		t.Fatalf("unexpected first break: from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
	br, found, err = NextParagraphBreak(c, 2, ParagraphByLineBreak)
	if err != nil {
		t.Fatalf("NextParagraphBreak(2) failed: %v", err)
	}
	if !found || br.AtByte != 3 || br.Length != 1 {
		t.Fatalf("unexpected second break: from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
	br, found, err = NextParagraphBreak(c, 4, ParagraphByLineBreak)
	if err != nil {
		t.Fatalf("NextParagraphBreak(4) failed: %v", err)
	}
	if found || br.AtByte != c.Len() || br.Length != 0 {
		t.Fatalf("expected no further break, got from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
}

func TestNextParagraphBreakCRLF(t *testing.T) {
	c := cordext.FromStringNoExt("A\r\nB")
	br, found, err := NextParagraphBreak(c, 0, ParagraphByLineBreak)
	if err != nil {
		t.Fatalf("NextParagraphBreak failed: %v", err)
	}
	if !found || br.AtByte != 1 || br.Length != 2 {
		t.Fatalf("unexpected CRLF break: from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
}

func TestNextParagraphBreakBlankLines(t *testing.T) {
	c := cordext.FromStringNoExt("A\n\nB\n\n\nC")
	br, found, err := NextParagraphBreak(c, 0, ParagraphByBlankLines)
	if err != nil {
		t.Fatalf("NextParagraphBreak(0, blank-lines) failed: %v", err)
	}
	if !found || br.AtByte != 1 || br.Length != 2 {
		t.Fatalf("unexpected first blank-line separator: from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
	br, found, err = NextParagraphBreak(c, 3, ParagraphByBlankLines)
	if err != nil {
		t.Fatalf("NextParagraphBreak(3, blank-lines) failed: %v", err)
	}
	if !found || br.AtByte != 4 || br.Length != 3 {
		t.Fatalf("unexpected second blank-line separator: from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
}

func TestNextParagraphBreakStandaloneCRIgnored(t *testing.T) {
	c := cordext.FromStringNoExt("A\rB\nC")

	br, found, err := NextParagraphBreak(c, 0, ParagraphByLineBreak)
	if err != nil {
		t.Fatalf("NextParagraphBreak failed: %v", err)
	}
	if !found || br.AtByte != 3 || br.Length != 1 {
		t.Fatalf("standalone CR should be ignored; expected LF break, got from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
}

func TestNextParagraphBreakCRLFAcrossChunkBoundary(t *testing.T) {
	prefix := strings.Repeat("x", chunk.MaxBase-1)
	c := cordext.FromStringNoExt(prefix + "\r\nZ")
	wantFrom := uint64(len(prefix))
	br, found, err := NextParagraphBreak(c, 0, ParagraphByLineBreak)
	if err != nil {
		t.Fatalf("NextParagraphBreak failed: %v", err)
	}
	if !found || br.AtByte != wantFrom || br.Length != 2 {
		t.Fatalf("unexpected cross-chunk CRLF break: from=%d length=%d found=%v", br.AtByte, br.Length, found)
	}
}

func TestForEachLineBreakFromMidChunkIncludesStartChunk(t *testing.T) {
	// Build 3 chunks:
	//   - chunk 0: all 'a'
	//   - chunk 1: contains one '\n' at local offset 10
	//   - chunk 2: no line breaks
	chunk0 := strings.Repeat("a", chunk.MaxBase)
	chunk1 := strings.Repeat("b", 10) + "\n" + strings.Repeat("c", chunk.MaxBase-11)
	chunk2 := "TAIL"
	c := cordext.FromStringNoExt(chunk0 + chunk1 + chunk2)

	start := uint64(chunk.MaxBase + 5)     // inside chunk 1, before '\n'
	wantFrom := uint64(chunk.MaxBase + 10) // '\n' in chunk 1

	var breaks []byteRange
	err := forEachLineBreak(c, start, func(br byteRange) bool {
		breaks = append(breaks, br)
		return false
	})
	if err != nil {
		t.Fatalf("forEachLineBreakNewlineBitmapFrom(mid-chunk) failed: %v", err)
	}
	if len(breaks) != 1 {
		t.Fatalf("expected exactly 1 line break, got %d", len(breaks))
	}
	if breaks[0].from != wantFrom || breaks[0].to != wantFrom+1 {
		t.Fatalf("expected LF break in same chunk at [%d,%d), got [%d,%d)",
			wantFrom, wantFrom+1, breaks[0].from, breaks[0].to)
	}
}

func TestNextParagraphBreakBounds(t *testing.T) {
	c := cordext.FromStringNoExt("abc")
	if _, _, err := NextParagraphBreak(c, 4, ParagraphByLineBreak); !errors.Is(err, cordext.ErrIndexOutOfBounds) {
		t.Fatalf("expected ErrIndexOutOfBounds, got %v", err)
	}
}
