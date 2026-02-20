package cords

import (
	"io"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestNewStringCord(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c := FromString("Hello World")
	t.Logf("c = '%s'", c)
	if c.String() != "Hello World" {
		t.Error("Expected cords.String() to be 'Hello World', is not")
	}
	if c.Len() != 11 {
		t.Errorf("Expected tree-backed cord len to be 11, is %d", c.Len())
	}
}

func TestCordIndex1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c := FromString("Hello World")
	leaf, i, err := c.Index(6)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("str[%d] = %c", i, leaf.String()[i])
	if leaf.String()[i] != 'W' || i != 6 {
		t.Error("expected index at 6/'W', isn't")
	}
}

func TestCordConcat(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello World")
	c2 := FromString(", how are you?")
	c := Concat(c1, c2)
	if c.IsVoid() {
		t.Fatal("concatenation is nil")
	}
	t.Logf("c = '%s'", c)
	if c.String() != "Hello World, how are you?" {
		t.Fatalf("unexpected concat result: %q", c.String())
	}
}

func TestCordLength1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello ")
	t.Logf("c1.len=%d", c1.Len())
	c2 := FromString("World")
	t.Logf("c2.len=%d", c2.Len())
	c := Concat(c1, c2)
	if c.Len() != 11 {
		t.Fatalf("expected length 11, got %d", c.Len())
	}
}

func TestCordSummaryCounts(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	c := FromString("a\nä\n")
	s := c.Summary()
	if s.Bytes != c.Len() {
		t.Fatalf("summary bytes mismatch: got=%d want=%d", s.Bytes, c.Len())
	}
	if s.Chars != c.CharCount() {
		t.Fatalf("summary chars mismatch: got=%d want=%d", s.Chars, c.CharCount())
	}
	if s.Lines != c.LineCount() {
		t.Fatalf("summary lines mismatch: got=%d want=%d", s.Lines, c.LineCount())
	}
	if s.Chars != 4 {
		t.Fatalf("unexpected char count: got=%d want=4", s.Chars)
	}
	if s.Lines != 2 {
		t.Fatalf("unexpected line count: got=%d want=2", s.Lines)
	}
}

func TestRotateLeft(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	t.Skip("legacy binary rotation helper removed")
}

func TestCordIndex2(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2)
	leaf, i, err := c.Index(6)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("str[%d] = %c", i, leaf.String()[i])
	if leaf.String()[i] != 'W' || i != 0 {
		t.Error("expected index at 0/'W', isn't")
	}
}

func TestBalance1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	t.Skip("legacy binary balance helper removed")
}

func TestCordSplit1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2) // now have a tree of height 3
	cl, cr, err := Split(c, 2)
	if err != nil {
		t.Fatal(err.Error())
	}
	if cl.IsVoid() || cr.IsVoid() {
		t.Fatal("Split resulted in empty partial cord, should not")
	}
	if cl.String() != "He" || cr.String() != "llo World" {
		t.Errorf("Expected split 'He'|'llo World', but left part is %v", cl)
	}
	if c.String() != "Hello World" {
		t.Fatalf("source cord changed after split: %q", c.String())
	}
}

func TestCordInsert(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2) // Hello World
	x, err := Insert(c, FromString(","), 5)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("x = '%s'", x.String())
	if x.Len() != 12 {
		t.Errorf("Expected result to be of length 12, is %d", x.Len())
	}
	x, err = Insert(x, FromString("!"), 12)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("x = '%s'", x.String())
	if x.String() != "Hello, World!" {
		t.Errorf("Double insert resulted in inexpected string: %s", x)
	}
}

func TestCordSplit2(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	// Build example cord from Wikipedia
	c3 := Concat(FromString("s"), FromString("_Simon"))
	c2 := Concat(FromString("na"), FromString("me_i"))
	c1 := Concat(FromString("Hello_"), FromString("my_"))
	c := Concat(c2, c3)
	c = Concat(c1, c)
	t.Logf("cord='%s'", c)
	//
	c1, c2, err := Split(c, 11)
	if err != nil {
		t.Fatal(err.Error())
	}
	if c1.String() != "Hello_my_na" || c1.Len() != 11 {
		t.Errorf("expected left cord to be 11 bytes, is: %s|%d", c1, c1.Len())
	}
	if c2.String() != "me_is_Simon" || c2.Len() != 11 {
		t.Errorf("expected right cord to be 11 bytes, is: %s|%d", c2, c2.Len())
	}
}

func TestCordCut(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2)       // Hello World
	x, y, err := Cut(c, 4, 4) // => Hellrld
	if err != nil {
		t.Fatal(err.Error())
	}
	if x.String() != "Hellrld" {
		t.Errorf("Expected cut-result to be 'Hellrld', is '%s'", x)
	}
	if y.String() != "o Wo" {
		t.Errorf("Expected cut-out segment to be 'o Wo', is '%s'", y)
	}
}

func TestCordReport(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c := FromString("Hello_")
	c = Concat(c, FromString("my_"))
	c = Concat(c, FromString("na"))
	c = Concat(c, FromString("me_i"))
	c = Concat(c, FromString("s"))
	c = Concat(c, FromString("_Simon"))
	t.Logf("cord='%s'", c)
	s, err := c.Report(8, 5)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("s='%s'", s)
	if s != "_name" {
		t.Errorf("Expected resulting string to be '_name', is '%s'", s)
	}
}

func TestCordSubstr(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	// Build example cord from Wikipedia
	c3 := Concat(FromString("s"), FromString("_Simon"))
	c2 := Concat(FromString("na"), FromString("me_i"))
	c1 := Concat(FromString("Hello_"), FromString("my_"))
	c := Concat(c2, c3)
	c = Concat(c1, c)
	t.Logf("cord='%s'", c)
	x, err := Substr(c, 8, 5)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("x='%s'", x)
	if x.String() != "_name" {
		t.Errorf("Expected resulting string to be '_name', is '%s'", x)
	}
	if x.Len() != 5 {
		t.Errorf("Expected substring length of 5, is %d", x.Len())
	}
}

func TestCordBuilder(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	b := NewBuilder()
	_ = b.AppendString("name_is")
	_ = b.PrependString("Hello_my_")
	_ = b.AppendString("_Simon")
	cord := b.Cord()
	if cord.IsVoid() {
		t.Fatal("Expected non-void result cord, is void")
	}
	t.Logf("builder made cord='%s'", cord)
	if cord.String() != "Hello_my_name_is_Simon" {
		t.Error("cord string is different from expected string")
	}
}

func TestTextSegmentIteration(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	c := FromString("Hello\nWorld")
	var got string
	var totalBytes uint64
	var prevPos uint64
	first := true
	err := c.EachTextSegment(func(seg TextSegment, pos uint64) error {
		if !first && pos < prevPos {
			t.Fatalf("segment positions must be monotonic: prev=%d now=%d", prevPos, pos)
		}
		first = false
		prevPos = pos
		if uint64(len(seg.Bytes())) != seg.ByteLen() {
			t.Fatalf("segment byte length mismatch: len(Bytes)=%d ByteLen=%d", len(seg.Bytes()), seg.ByteLen())
		}
		got += seg.String()
		totalBytes += seg.ByteLen()
		return nil
	})
	if err != nil {
		t.Fatalf("EachTextSegment failed: %v", err)
	}
	if got != c.String() {
		t.Fatalf("segment concat mismatch: got=%q want=%q", got, c.String())
	}
	if totalBytes != c.Len() {
		t.Fatalf("segment byte sum mismatch: got=%d want=%d", totalBytes, c.Len())
	}
}

func TestCordCutAndInsert(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	b := NewBuilder()
	_ = b.AppendString("Hello_")
	_ = b.AppendString("my_")
	_ = b.AppendString("name_")
	_ = b.AppendString("is")
	_ = b.AppendString("_Simon")
	c := b.Cord()
	x, _, err := Cut(c, 10, 5)
	//c, _, err := Split(c, 10)
	if err != nil {
		t.Fatal(err.Error())
	}
	l := FromString("THIS IS NEW")
	y, _ := Insert(x, l, 10)
	if y.IsVoid() {
		t.Error("cord is void after cut and insert")
	}
}

func TestCordReader(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	b := NewBuilder()
	_ = b.AppendString("name_is")
	_ = b.PrependString("Hello_my_")
	_ = b.AppendString("_Simon")
	cord := b.Cord()
	if cord.IsVoid() {
		t.Fatalf("Expected non-void result cord, is void")
	}
	t.Logf("builder made cord='%s'", cord)
	reader := cord.Reader()
	p := make([]byte, 5)
	n, err := reader.Read(p)
	if err != nil {
		t.Error(err.Error())
	}
	if n != 5 || string(p) != "Hello" {
		t.Logf("n=%d, p=%s", n, string(p))
		t.Fatal("expected Read() to return 'Hello', did not")
	}
	n, err = reader.Read(p)
	if err != nil {
		t.Error(err.Error())
	}
	if n != 5 {
		t.Logf("n=%d, p=%s", n, string(p))
		t.Fatalf("expected Read() to return 5 bytes, have %d", n)
	}
	p = make([]byte, 50)
	n, err = reader.Read(p)
	if err != nil {
		t.Error(err.Error())
	}
	if n != 12 {
		t.Logf("n=%d, p=%s", n, string(p))
		t.Fatalf("expected Read() to return 12 bytes, have %d", n)
	}
	_, err = reader.Read(p)
	if err != io.EOF {
		if err != nil {
			t.Error(err.Error())
		} else {
			t.Error("expected EOF, got no error")
		}
	}
}

func TestRangeIterator(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	b := NewBuilder()
	_ = b.AppendString("name_is")
	_ = b.PrependString("Hello_my_")
	_ = b.AppendString("_Simon")
	cord := b.Cord()
	if cord.IsVoid() {
		t.Fatalf("Expected non-void result cord, is void")
	}
	s := strings.Builder{}
	for c := range cord.RangeChunk() {
		s.WriteString(c.String())
	}
	if s.String() != "Hello_my_name_is_Simon" {
		t.Fatalf("expected 'Hello_my_name_is_Simon', got '%s'", s.String())
	}
}
