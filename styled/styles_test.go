package styled

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/npillmayer/cords"

	//"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestInsert(t *testing.T) {
	cb := cords.NewBuilder()
	bold := teststyle("bold")
	spn := toSpan(1, 5)
	cb.Append(makeStyleLeaf(bold, spn))
	_ = cb.Cord()
}

func TestBasicStyle(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	// make a text
	text := cords.FromString("Hello World")
	t.Logf("string='%s', length=%d", text, text.Len())
	// style the text
	bold := teststyle("bold")
	runs := applyStyle(text, bold, 6, text.Len())
	t.Logf("runs=%s, length=%d", runs.String(), runs.Len())
	//
	cnt := cords.Cord(runs).FragmentCount()
	if cnt != 2 {
		t.Errorf("expected formatted text to have 2 segments, has %d", cnt)
	}
	runs = runs.Style(bold, 0, 1)
	cnt = cords.Cord(runs).FragmentCount()
	if cnt != 3 {
		t.Errorf("expected formatted text to have 3 segments, has %d", cnt)
	}
}

func TestTextSimple(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	text := TextFromString("Hello World, how are you?")
	bold, italic := teststyle("bold"), teststyle("italic")
	text.Style(bold, 6, 11)
	text.Style(italic, 8, 16) // erase part of bold run
	cnt := cords.Cord(text.styles()).FragmentCount()
	if cnt != 4 {
		t.Errorf("expected formatted text to have 4 segments, has %d", cnt)
	}
}

func TestEach(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	text := TextFromString("Hello World, how are you?")
	bold := teststyle("bold")
	text.Style(bold, 6, 16)
	//
	cnt := 0
	text.EachStyleRun(func(content string, sty Style, pos uint64) error {
		cnt++
		t.Logf("%v: (%s)", sty, content)
		return nil
	})
	if cnt != 3 {
		t.Errorf("expected formatted text to have 3 style runs, has %d", cnt)
	}
}

// --- Test Helpers ----------------------------------------------------------

type mystyle []string

func teststyle(sty string) mystyle {
	return mystyle{sty}
}

func (sty mystyle) Equals(other Style) bool {
	o := other.(mystyle)
	if len(sty) != len(o) {
		return false
	}
	for i, s := range o {
		if s != sty[i] {
			return false
		}
	}
	return true
}

func (sty mystyle) String() string {
	return fmt.Sprintf("%v", []string(sty))
}

var _ Style = mystyle{}

type testfmtr struct {
	segcnt int
	out    *bytes.Buffer
}

func formatter(prefix string) *testfmtr {
	return &testfmtr{
		out: bytes.NewBufferString(prefix),
	}
}

func (vf testfmtr) String() string {
	return vf.out.String()
}

func (vf *testfmtr) StartRun(f Style, w io.Writer) error {
	vf.segcnt++
	if f == nil {
		_, err := w.Write([]byte("[plain]"))
		return err
	}
	sty := f.(mystyle)
	_, err := w.Write([]byte(sty.String()))
	return err
}

func (vf testfmtr) Format(buf []byte, f Style, w io.Writer) error {
	w.Write(buf)
	return nil
}

func (vf testfmtr) EndRun(f Style, w io.Writer) error {
	_, err := w.Write([]byte("|"))
	return err
}
