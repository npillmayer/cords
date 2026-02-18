package textfile

import (
	"sync"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestLoad(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	var wg sync.WaitGroup
	cord, err := Load("lorem/lorem_small.txt", 0, 0, &wg)
	if err != nil {
		t.Fatal(err.Error())
	}
	if cord.IsVoid() {
		t.Errorf("cord is void, should not be")
	}
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
