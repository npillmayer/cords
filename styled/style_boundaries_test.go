package styled

import "testing"

func TestStyleAtTextBoundaries(t *testing.T) {
	text := TextFromString("é אב z")
	original := text
	for _, span := range []struct {
		from, to uint64
		name     string
	}{{0, 3, "a"}, {3, 7, "b"}, {7, 9, "a"}, {0, 9, "c"}} {
		var err error
		text, err = text.Style(teststyle(span.name), span.from, span.to)
		if err != nil {
			t.Fatal(err)
		}
		if got, _, _ := text.StyleAt(span.from); !equals(got, teststyle(span.name)) {
			t.Fatalf("wrong style: %v", got)
		}
	}
	if got, _, _ := original.StyleAt(0); got != nil {
		t.Fatal("original mutated")
	}
}
