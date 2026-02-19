package chunk

import "testing"

func TestChunkSummaryCounts(t *testing.T) {
	c, err := New("a\n😀b")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	s := c.Summary()
	if s.Bytes != 7 || s.Chars != 4 || s.Lines != 1 {
		t.Fatalf("unexpected summary: %+v", s)
	}
}

func TestChunkSliceSummaryCounts(t *testing.T) {
	c, err := New("a\n😀b")
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	sl, err := c.Slice(1, 6) // "\n😀"
	if err != nil {
		t.Fatalf("unexpected Slice error: %v", err)
	}
	s := sl.Summary()
	if s.Bytes != 5 || s.Chars != 2 || s.Lines != 1 {
		t.Fatalf("unexpected slice summary: %+v", s)
	}
}

func TestSummaryMonoid(t *testing.T) {
	a := Summary{Bytes: 5, Chars: 3, Lines: 1}
	b := Summary{Bytes: 4, Chars: 2, Lines: 0}
	m := Monoid{}
	c := m.Add(a, b)
	if c.Bytes != 9 || c.Chars != 5 || c.Lines != 1 {
		t.Fatalf("unexpected monoid add result: %+v", c)
	}
	if z := m.Zero(); z != (Summary{}) {
		t.Fatalf("unexpected monoid zero value: %+v", z)
	}
}
