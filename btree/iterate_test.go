package btree

import (
	"strconv"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func buildTextTree(t *testing.T, n int) *Tree[textChunk, textSummary, NO_EXT] {
	t.Helper()
	tree, err := New(Config[textChunk, textSummary, NO_EXT]{
		Monoid: textMonoid{},
	})
	if err != nil {
		t.Fatalf("new tree failed: %v", err)
	}
	for i := range n {
		tree, err = tree.InsertAt(tree.Len(), fromString(strconv.Itoa(i)))
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}
	return tree
}

func collectStrings(items []textChunk) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = string(item)
	}
	return out
}

func TestItemRange(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	tree := buildTextTree(t, 10)
	got := make([]string, 0, 4)
	for item, _ := range tree.ItemRange(3, 7) {
		got = append(got, item.String())
	}
	want := []string{"3", "4", "5", "6"}
	if len(got) != len(want) {
		t.Fatalf("range length mismatch: got=%d want=%d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("range mismatch at %d: got=%v want=%v", i, got, want)
		}
	}
}

func collectItemRange(tree *Tree[textChunk, textSummary, NO_EXT], from, to int64) (
	[]string, []int64) {
	//
	items := make([]string, 0, max(0, to-from))
	indexes := make([]int64, 0, max(0, to-from))
	for item, idx := range tree.ItemRange(from, to) {
		items = append(items, item.String())
		indexes = append(indexes, idx)
	}
	return items, indexes
}

func TestItemRangeEmptyAndSingle(t *testing.T) {
	tree := buildTextTree(t, 10)

	items, indexes := collectItemRange(tree, 4, 4)
	if len(items) != 0 || len(indexes) != 0 {
		t.Fatalf("empty range should yield no items, got items=%v idx=%v", items, indexes)
	}

	items, indexes = collectItemRange(tree, 0, 1)
	if len(items) != 1 || items[0] != "0" {
		t.Fatalf("single-item range [0,1) mismatch: %v", items)
	}
	if len(indexes) != 1 || indexes[0] != 0 {
		t.Fatalf("single-item range [0,1) index mismatch: %v", indexes)
	}

	last := tree.Len() - 1
	items, indexes = collectItemRange(tree, last, tree.Len())
	if len(items) != 1 || items[0] != strconv.Itoa(int(last)) {
		t.Fatalf("single-item tail range mismatch: %v", items)
	}
	if len(indexes) != 1 || indexes[0] != last {
		t.Fatalf("single-item tail index mismatch: %v", indexes)
	}
}

func TestItemRangeCrossLeafAndIndexes(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	tree := buildTextTree(t, 60)
	var from, to int64 = 10, 19
	items, indexes := collectItemRange(tree, from, to)
	if len(items) != int(to-from) {
		t.Fatalf("range length mismatch: got=%d want=%d (%v)", len(items), to-from, items)
	}
	for i := from; i < to; i++ {
		slot := i - from
		wantItem := strconv.Itoa(int(i))
		if items[slot] != wantItem {
			t.Fatalf("range item mismatch at %d: got=%v want=%v", slot, items, wantItem)
		}
		if indexes[slot] != i {
			t.Fatalf("range index mismatch at %d: got=%v want=%d", slot, indexes, i)
		}
	}
}

func TestItemRangeLastThreeLargeTree(t *testing.T) {
	tree := buildTextTree(t, 1000)
	from := tree.Len() - 3
	items, indexes := collectItemRange(tree, from, tree.Len())
	wantItems := []string{"997", "998", "999"}
	wantIdx := []int64{997, 998, 999}
	if len(items) != len(wantItems) {
		t.Fatalf("tail range length mismatch: got=%d want=%d (%v)", len(items), len(wantItems), items)
	}
	for i := range wantItems {
		if items[i] != wantItems[i] {
			t.Fatalf("tail range item mismatch: got=%v want=%v", items, wantItems)
		}
		if indexes[i] != wantIdx[i] {
			t.Fatalf("tail range index mismatch: got=%v want=%v", indexes, wantIdx)
		}
	}
}
