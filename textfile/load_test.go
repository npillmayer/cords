package textfile

import (
	"sync"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"

	//"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/schuko/tracing/gologadapter"
)

func TestLoad(t *testing.T) {
	gtrace.CoreTracer = gologadapter.New()
	//teardown := gotestingadapter.RedirectTracing(t)
	//defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var wg sync.WaitGroup
	cord, start, err := Load("lorem/lorem_small.txt", 0, 0, &wg)
	if err != nil {
		t.Fatal(err.Error())
	}
	if cord.IsVoid() {
		t.Errorf("cord is void, should not be")
	}
	ext := start.Ext().loadExt
	t.Logf("start leaf %d…%d", ext.pos, start.Weight())
	t.Logf("done waiting -----------------------------------------------------------")
	// time.Sleep(2 * time.Second)
	s, err := cord.Report(0, 40)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("cord starts with '%s'…", s)
	wg.Wait()
	//t.Fail()
}
