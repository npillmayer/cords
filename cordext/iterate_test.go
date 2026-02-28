package cordext

import (
	"slices"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

const lorem = `Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod
tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero
eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea
takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur
sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam
erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet
clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet.`

const newlineCount = 6

func TestSegmentRangeBounded(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.cords")
	defer teardown()

	text := lorem
	cord, err := FromStringWithExtension(text, newlineExt{})
	if err != nil {
		t.Fatalf("FromStringWithExtension failed: %v", err)
	}
	ext, ok := cord.Ext()
	if !ok {
		t.Fatalf("Ext() not available")
	}
	if ext != newlineCount {
		t.Fatalf("newline extension mismatch: got=%d want=%d", ext, newlineCount)
	}
	for seg := range cord.RangeTextSegment() {
		nl := seg.Newlines() != 0
		t.Logf("1. (nl=%v) @ %s", nl, seg.String())
	}
	t.Log("-----------------------------------------------------------")
	count := 0
	for seg := range cord.TextSegmentRangeBounded(1, 2) {
		nl := seg.Newlines() != 0
		t.Logf("2. [nl=%v] @ %s", nl, seg.String())
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 segment, got %d", count)
	}
}

func TestByteRangeBoundedWithinChunk(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.cords")
	defer teardown()

	text := lorem
	cord, err := FromStringWithExtension(text, newlineExt{})
	if err != nil {
		t.Fatalf("FromStringWithExtension failed: %v", err)
	}
	count := 0 // counting result bytes
	for i, b := range cord.ByteRangeBounded(6, 11) {
		t.Logf("[%d] @ %v ('%c')", i, b, b)
		count++
	}
	if count != 5 {
		t.Fatalf("expected 5 bytes, got %d", count)
	}
}

func TestByteRangeBoundedSpanningChunks(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.cords")
	defer teardown()

	text := lorem
	cord, err := FromStringWithExtension(text, newlineExt{})
	if err != nil {
		t.Fatalf("FromStringWithExtension failed: %v", err)
	}
	var seglen [2]int // find out length of first 2 chunks
	for i, chunk := range cord.ChunkRangeBounded(0, 2) {
		seglen[i] = chunk.Len()
		t.Logf("#%d/%d : %s", i, seglen[i], chunk.String())
	}
	count := 0 // counting result bytes
	endOfChunk := uint64(seglen[0])
	for i, b := range cord.ByteRangeBounded(endOfChunk-5, endOfChunk+5) {
		t.Logf("[%d] @ %v ('%c')", i, b, b)
		count++
	}
	if count != 10 {
		t.Fatalf("expected 10 bytes, got %d", count)
	}
}

func TestByteBoundedReader(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.cords")
	defer teardown()

	text := lorem
	cord, err := FromStringWithExtension(text, newlineExt{})
	if err != nil {
		t.Fatalf("FromStringWithExtension failed: %v", err)
	}
	reader := cord.BoundedReader(0, 10)
	buf := make([]byte, 10)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	t.Logf("cords.cordext reader got %q", buf)
	if n != 9 {
		t.Fatalf("expected 9 bytes, got %d", n)
	}
	if string(buf) != text[:10] {
		t.Fatalf("expected %q, got %q", text[:10], buf)
	}
}

func TestByteReader(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords.cords")
	defer teardown()
	//
	var (
		err  error
		cord CordEx[uint64]
		n    int
	)
	text := "The quick brown fox jumps over the lazy dog."
	cord, err = FromStringWithExtension(text, newlineExt{})
	if err != nil {
		t.Fatalf("FromStringWithExtension failed: %v", err)
	}
	buf := make([]byte, 10)
	total := make([]byte, 0, len(text))
	reader := cord.Reader()
	safety := 0
	for err == nil && safety < 10 {
		n, err = reader.Read(buf)
		if n > 0 {
			t.Logf("Read %d bytes: %q", n, buf[:n])
		}
		total = append(total, buf[:n]...)
		safety++
	}
	if safety == 0 {
		t.Fatalf("reader did not read in batches")
	}
	if slices.Compare([]byte(text), total) != 0 {
		t.Fatalf("expected %q, got %q", text, total)
	}
}
