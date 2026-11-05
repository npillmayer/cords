package btree

import "testing"

func TestItemRangeHonorsEarlyStop(t *testing.T) {
	for _, size := range []int{10, 60, 1000} {
		tree := buildTextTree(t, size)
		for _, stopAfter := range []int{1, 5, size / 2} {
			seq := tree.ItemRange(0, tree.Len())
			calls := 0
			seq(func(_ int64, _ textChunk) bool {
				calls++
				return calls < stopAfter
			})
			if calls != stopAfter {
				t.Fatalf("size=%d: called yield %d times, want %d", size, calls, stopAfter)
			}
			// A new traversal starts fresh, including after a range-loop break.
			for range seq {
				break
			}
			calls = 0
			for range seq {
				calls++
			}
			if calls != size {
				t.Fatal("iterator not reusable", calls, size)
			}
		}
	}
}
