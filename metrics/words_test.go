package metrics

import (
	"testing"

	"github.com/npillmayer/cords"
)

func TestWordsApplyWholeCord(t *testing.T) {
	c := cords.FromString("Hello  my\nname\tis Simon")

	value, materialized, err := Words().Apply(c, 0, c.Len())
	if err != nil {
		t.Fatalf("Words().Apply failed: %v", err)
	}
	if value.WordCount() != 5 {
		t.Fatalf("unexpected word count: got=%d want=5", value.WordCount())
	}
	if materialized.String() != "HellomynameisSimon" {
		t.Fatalf("unexpected materialized text: got=%q", materialized.String())
	}
	want := []Span{
		{Pos: 0, Len: 5},
		{Pos: 7, Len: 2},
		{Pos: 10, Len: 4},
		{Pos: 15, Len: 2},
		{Pos: 18, Len: 5},
	}
	if len(value.Spans) != len(want) {
		t.Fatalf("unexpected spans len: got=%d want=%d", len(value.Spans), len(want))
	}
	for i := range want {
		if value.Spans[i] != want[i] {
			t.Fatalf("span %d mismatch: got=%+v want=%+v", i, value.Spans[i], want[i])
		}
	}
}

func TestWordsApplySubrange(t *testing.T) {
	c := cords.FromString("xx Hello world yy")
	// "Hello world"
	value, materialized, err := Words().Apply(c, 3, 14)
	if err != nil {
		t.Fatalf("Words().Apply failed: %v", err)
	}
	if value.WordCount() != 2 {
		t.Fatalf("unexpected word count: got=%d want=2", value.WordCount())
	}
	if len(value.Spans) != 2 {
		t.Fatalf("unexpected spans len: got=%d want=2", len(value.Spans))
	}
	if value.Spans[0] != (Span{Pos: 3, Len: 5}) {
		t.Fatalf("first span mismatch: got=%+v", value.Spans[0])
	}
	if value.Spans[1] != (Span{Pos: 9, Len: 5}) {
		t.Fatalf("second span mismatch: got=%+v", value.Spans[1])
	}
	if materialized.String() != "Helloworld" {
		t.Fatalf("unexpected materialized text: got=%q", materialized.String())
	}
}

func TestWordsApplyBoundsValidation(t *testing.T) {
	c := cords.FromString("abc")
	_, _, err := Words().Apply(c, 2, 1)
	if err == nil {
		t.Fatalf("expected error for invalid range")
	}
}
