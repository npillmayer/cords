package cords

import (
	"io"
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
	if c.root.height != 2 {
		t.Errorf("Height of root = %d, should be 2", c.root.height)
	}
	leaf := c.root.left
	if !leaf.IsLeaf() {
		t.Error("expected leaf at height 1, is not")
	}
}

func TestCordIndex1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c := FromString("Hello World")
	node, i, err := c.index(6)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("str[%d] = %c", i, node.String()[i])
	if node.String()[i] != 'W' || i != 6 {
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
	t.Logf("c.left = '%s'", c.root.Left())
	t.Logf("c.right = '%s'", c.root.Right())
	if c.root.height != 3 {
		t.Errorf("expected height(c) to be 3, is %d", c.root.Height())
	}
	if &c1.root.cordNode == c.root.left {
		t.Error("copy on write did not work for c1.root")
	}
}

func TestCordLength1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello ")
	t.Logf("c1.len=%d", c1.root.Len())
	c2 := FromString("World")
	t.Logf("c2.len=%d", c2.root.Len())
	c := Concat(c1, c2)
	if c.root.Len() != c.Len() {
		t.Errorf("length calculation of top inner node failed, %d != %d", c.root.Len(), c.Len())
	}
	if c.root.left.Len() != c.Len() {
		t.Logf("w=%d", c.root.left.Weight())
		t.Errorf("length calculation of inner node failed, %d != %d", c.root.left.Len(), c.Len())
	}
	if c.root.Len() != 11 || c.root.Left().Len() != 11 {
		t.Errorf("length calculation is off, expected 11, is %d|%d", c.root.Len(), c.root.Left().Len())
	}
}

func TestRotateLeft(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello")
	c2 := FromString(" World,")
	c3 := FromString(", how are you?")
	c := Concat(c1, c2)
	dump(&c.root.cordNode)
	t.Logf("-----------------------------------------------")
	c = Concat(c, c3)
	dump(&c.root.cordNode)
	t.Logf("-----------------------------------------------")
	x := rotateRight(c.root)
	dump(&x.cordNode)
	if x.Left().Height() != 2 || x.Right().Height() != 2 {
		t.Error("Expected both left and right sub-tree to be of height 2, aren't")
	}
}

func TestCordIndex2(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello ")
	c2 := FromString("World")
	c := Concat(c1, c2)
	node, i, err := c.index(6)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("str[%d] = %c", i, node.String()[i])
	if node.String()[i] != 'W' || i != 0 {
		t.Error("expected index at 0/'W', isn't")
	}
}

func TestBalance1(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	c1 := FromString("Hello")
	c2 := FromString(" World,")
	c3 := FromString(", how are")
	c4 := FromString("you?")
	c := Concat(c1, c2)
	c = Concat(c, c3)
	c = Concat(c, c4)
	ub := unbalanced(c.root.left)
	t.Logf("balance of c = %v", !ub)
	t.Logf("height of left = %d", c.root.Left().Height())
	if ub {
		dump(&c.root.cordNode)
		t.Error("cord tree not balanced after multiple concatenations")
	}
	if c.root.leftHeight() != 3 {
		dump(&c.root.cordNode)
		t.Error("cord tree too high after multiple concatenations")
		top := c.root.left.AsInner()
		t.Logf("top=%v, l.h=%d, r.h=%d", top, top.leftHeight(), top.rightHeight())
	}
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
	t.Logf("----------------------")
	dump(&cl.root.cordNode)
	t.Logf("----------------------")
	dump(&cr.root.cordNode)
	t.Logf("======================")
	dump(&c.root.cordNode)
	if cl.root == nil || cr.root == nil {
		t.Fatal("Split resulted in empty partial cord, should not")
	}
	if cl.root.Height() != 2 || cr.root.Height() != 3 {
		t.Errorf("Expected split sub-trees of height 2 and 3, are %d and %d", cl.root.Height(), cr.root.Height())
	}
	if cl.String() != "He" || cr.String() != "llo World" {
		t.Errorf("Expected split 'He'|'llo World', but left part is %v", cl)
	}
	if c.root == cl.root || c.root == cr.root {
		t.Fatal("copy on write did not work as expected")
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
	dump(&c.root.cordNode)
	//
	c1, c2, err := Split(c, 11)
	if err != nil {
		t.Fatal(err.Error())
	}
	if c1.String() != "Hello_my_na" || c1.Len() != 11 {
		dump(&c.root.cordNode)
		t.Logf("--------------------")
		dump(&c1.root.cordNode)
		t.Errorf("expected left cord to be 11 bytes, is: %s|%d", c1, c1.Len())
	}
	if c2.String() != "me_is_Simon" || c2.Len() != 11 {
		dump(&c2.root.cordNode)
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
	dump(&x.root.cordNode)
	t.Logf("x='%s'", x)
	if x.String() != "_name" {
		t.Errorf("Expected resulting string to be '_name', is '%s'", x)
	}
	if x.root.Weight() != 5 {
		t.Errorf("Expected weight of root node to be |_name|=5, is %d", x.root.Weight())
	}
	if x.root.Height() != 4 {
		t.Errorf("Expected height of root node to be 4, is %d", x.root.Height())
	}
}

func TestCordBuilder(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	b := NewBuilder()
	b.Append(StringLeaf("name_is"))
	b.Prepend(StringLeaf("Hello_my_"))
	b.Append(StringLeaf("_Simon"))
	cord := b.Cord()
	if cord.IsVoid() {
		t.Fatal("Expected non-void result cord, is void")
	}
	dump(&cord.root.cordNode)
	t.Logf("builder made cord='%s'", cord)
	if cord.String() != "Hello_my_name_is_Simon" {
		t.Error("cord string is different from expected string")
	}
}

func TestCordCutAndInsert(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	b := NewBuilder()
	b.Append(StringLeaf("Hello_"))
	b.Append(StringLeaf("my_"))
	b.Append(StringLeaf("name_"))
	b.Append(StringLeaf("is"))
	b.Append(StringLeaf("_Simon"))
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
	b.Append(StringLeaf("name_is"))
	b.Prepend(StringLeaf("Hello_my_"))
	b.Append(StringLeaf("_Simon"))
	cord := b.Cord()
	if cord.IsVoid() {
		t.Fatalf("Expected non-void result cord, is void")
	}
	dump(&cord.root.cordNode)
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
			t.Error("exptected EOF, got no error")
		}
	}
}
