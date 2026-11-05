package cordext

import (
	"io"
	"strings"
	"testing"

	"github.com/npillmayer/cords/btree"
)

func TestBoundedBytesMatchOriginalAcrossChunks(t *testing.T) {
	source := strings.Repeat("abc é אב xyz ", 50)
	cord := FromStringNoExt(source)
	if cord.tree.Len() < 3 {
		t.Fatal("fixture must span multiple chunks")
	}
	// Exercise every starting byte, including continuation bytes and exact
	// chunk edges. Byte ranges deliberately do not require rune boundaries.
	for from := 0; from <= len(source)+1; from++ {
		for _, to := range []int{from, from + 1, from + 7, from + len(source)/2, len(source) + 10} {
			end := min(to, len(source))
			want := ""
			if from < end {
				want = source[from:end]
			}
			var got []byte
			for at, b := range cord.ByteRangeBounded(uint64(from), uint64(to)) {
				if at != from+len(got) {
					t.Fatalf("range %d..%d: absolute index %d, want %d", from, to, at, from+len(got))
				}
				got = append(got, b)
			}
			if string(got) != want {
				t.Fatalf("range %d..%d: got %q, want %q", from, to, got, want)
			}
		}
	}
	seq := cord.ByteRangeBounded(20, cord.Len()-20)
	for range seq {
		break // early termination must not alter the next traversal
	}
	for range 2 {
		var got []byte
		for _, b := range seq {
			got = append(got, b)
		}
		if string(got) != source[20:len(source)-20] {
			t.Fatal("iterator not reusable")
		}
	}
}

func TestBoundedReaderCountsAndEOF(t *testing.T) {
	source := strings.Repeat("alpha βeta ", 80)
	cord := FromStringNoExt(source)
	for _, bounds := range [][2]uint64{{0, 1}, {0, 10}, {9, 200}, {180, 181}, {200, cord.Len()}, {0, cord.Len()}} {
		want := source[bounds[0]:bounds[1]]
		for _, size := range []int{1, 2, 7, 10, 64, 4096} {
			r := cord.BoundedReader(bounds[0], bounds[1])
			buf := make([]byte, size)
			var got []byte
			for {
				if n, err := r.Read(nil); n != 0 || err != nil {
					t.Fatalf("empty read: %d %v", n, err)
				}
				n, err := r.Read(buf)
				if n < 0 || n > size || (n == 0 && err == nil) {
					t.Fatalf("invalid read count: %d %v", n, err)
				}
				got = append(got, buf[:n]...)
				if len(got) > len(want) {
					t.Fatal("read beyond bounds")
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if string(got) != want {
				t.Fatalf("bounds=%v size=%d: got %q, want %q", bounds, size, got, want)
			}
			if n, err := r.Read(buf); n != 0 || err != io.EOF {
				t.Fatalf("repeated EOF: %d %v", n, err)
			}
		}
	}
}

func TestBoundedReaderEmptyRanges(t *testing.T) {
	for _, cord := range []CordEx[btree.NO_EXT]{{}, FromStringNoExt(""), FromStringNoExt("abc")} {
		for _, bounds := range [][2]uint64{{0, 0}, {2, 1}, {3, 3}, {10, 20}} {
			r := cord.BoundedReader(bounds[0], bounds[1])
			if r == nil {
				t.Fatal("empty range must return an EOF reader")
			}
			if got, err := io.ReadAll(r); err != nil || len(got) != 0 {
				t.Fatal(got, err)
			}
		}
	}
}
