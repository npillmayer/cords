package textfile

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/npillmayer/cords/chunk"
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

func TestLoadUTF8AcrossReadBoundaries(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	dir := t.TempDir()
	path := filepath.Join(dir, "utf8.txt")
	want := "aä🙂\nβ"
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	cord, err := Load(path, 0, 1, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := cord.String(); got != want {
		t.Fatalf("unexpected content: got=%q want=%q", got, want)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	cord, err := Load(path, 0, 0, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cord.IsVoid() {
		t.Fatalf("expected empty cord for empty file")
	}
}

func TestLoadRejectsInvalidUTF8(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad-utf8.txt")
	// Invalid UTF-8 byte sequence.
	if err := os.WriteFile(path, []byte{0xff, 0xfe, 'a'}, 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	_, err := Load(path, 0, 2, nil)
	if err == nil {
		t.Fatalf("expected UTF-8 validation error")
	}
	if !errors.Is(err, chunk.ErrInvalidUTF8) {
		t.Fatalf("expected chunk.ErrInvalidUTF8, got %v", err)
	}
}
