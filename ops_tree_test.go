package cords

import "testing"

func mustTreeCord(t *testing.T, s string) Cord {
	t.Helper()
	b := NewBuilder()
	if err := b.AppendString(s); err != nil {
		t.Fatalf("AppendString failed: %v", err)
	}
	c := b.Cord()
	if c.tree == nil {
		t.Fatalf("expected tree-backed cord")
	}
	return c
}

func TestTreePathSplitInsertCut(t *testing.T) {
	c := mustTreeCord(t, "Hello World")

	left, right, err := Split(c, 5)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}
	if left.tree == nil || right.tree == nil {
		t.Fatalf("expected split result to remain tree-backed")
	}
	if left.String() != "Hello" || right.String() != " World" {
		t.Fatalf("unexpected split result: %q | %q", left.String(), right.String())
	}

	inserted, err := Insert(c, FromString(","), 5)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if inserted.tree == nil {
		t.Fatalf("expected insert result to remain tree-backed")
	}
	if inserted.String() != "Hello, World" {
		t.Fatalf("unexpected insert result: %q", inserted.String())
	}

	cut, mid, err := Cut(inserted, 5, 2) // remove ", "
	if err != nil {
		t.Fatalf("Cut failed: %v", err)
	}
	if cut.String() != "HelloWorld" || mid.String() != ", " {
		t.Fatalf("unexpected cut result: %q / %q", cut.String(), mid.String())
	}
}

func TestTreePathReportAndReader(t *testing.T) {
	c := mustTreeCord(t, "Hello World")
	s, err := c.Report(6, 5)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}
	if s != "World" {
		t.Fatalf("unexpected report result: %q", s)
	}

	p := make([]byte, 5)
	n, err := c.Reader().Read(p)
	if err != nil {
		t.Fatalf("Reader.Read failed: %v", err)
	}
	if n != 5 || string(p) != "Hello" {
		t.Fatalf("unexpected read result: n=%d p=%q", n, string(p))
	}
}
