package styled

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	//"github.com/npillmayer/schuko/tracing/gologadapter"
)

func TestBasicStyle(t *testing.T) {
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	// make a text
	text := cords.FromString("Hello World")
	t.Logf("string='%s', length=%d", text, text.Len())
	// style the text
	bold := teststyle("bold")
	runs := Apply(text, bold, 6, text.Len())
	t.Logf("runs=%s, length=%d", runs.String(), runs.Len())
	// get formatted text
	s := bufio.NewScanner(strings.NewReader(text.String()))
	fmtr := formatter("Test BasicStyle: ")
	if err := runs.Format(s, fmtr, fmtr.out); err != nil {
		t.Error(err.Error())
	}
	t.Logf(fmtr.String())
	if fmtr.segcnt != 2 {
		t.Errorf("expected formatted text to have 2 segments, has %d", fmtr.segcnt)
	}
}

// --- Test Helpers ----------------------------------------------------------

type mystyle []string

func teststyle(sty string) mystyle {
	return mystyle{sty}
}

func (sty mystyle) Equals(other Format) bool {
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

var _ Format = mystyle{}

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

func (vf *testfmtr) StartRun(f Format, w io.Writer) error {
	vf.segcnt++
	if f == nil {
		_, err := w.Write([]byte("[plain]"))
		return err
	}
	sty := f.(mystyle)
	_, err := w.Write([]byte(sty.String()))
	return err
}

func (vf testfmtr) Format(buf []byte, f Format, w io.Writer) error {
	w.Write(buf)
	return nil
}

func (vf testfmtr) EndRun(f Format, w io.Writer) error {
	_, err := w.Write([]byte("|"))
	return err
}
