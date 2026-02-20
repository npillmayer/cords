package btree

import (
	"math/rand"
	"strconv"
	"testing"
)

// How to run:
//   - Deterministic randomized property test:
//     go test ./btree -run TestExtensionRandomizedProperty -count=1
//   - Fuzz test for this file:
//     go test ./btree -run '^$' -fuzz FuzzExtensionRandomizedProperty -fuzztime=10s
//   - Replay a specific saved failing input:
//     go test ./btree -run 'FuzzExtensionRandomizedProperty/<id>'

func newExtTextTree(t *testing.T, id string) *Tree[TextChunk, TextSummary, uint64] {
	t.Helper()
	tree, err := New[TextChunk, TextSummary](Config[TextChunk, TextSummary, uint64]{
		Monoid:    TextMonoid{},
		Extension: countingExt{id: id},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return tree
}

func randomToken(r *rand.Rand) string {
	n := r.Intn(4) + 1
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + r.Intn(26))
	}
	return string(b)
}

func expectedBytesFromModel(model []string) uint64 {
	var sum uint64
	for _, s := range model {
		sum += uint64(len(s))
	}
	return sum
}

func assertTreeMatchesModelAndExtension(t *testing.T, tree *Tree[TextChunk, TextSummary, uint64], model []string) {
	t.Helper()

	got := collectTextItems(tree)
	if len(got) != len(model) {
		t.Fatalf("model length mismatch: got=%d want=%d", len(got), len(model))
	}
	for i := range model {
		if got[i] != model[i] {
			t.Fatalf("model mismatch at %d: got=%q want=%q", i, got[i], model[i])
		}
	}

	wantBytes := expectedBytesFromModel(model)
	if tree.Summary().Bytes != wantBytes {
		t.Fatalf("summary bytes mismatch: got=%d want=%d", tree.Summary().Bytes, wantBytes)
	}

	prefix, err := tree.PrefixExt(len(model))
	if err != nil {
		t.Fatalf("PrefixExt(len) failed: %v", err)
	}
	if prefix != wantBytes {
		t.Fatalf("PrefixExt(len) mismatch: got=%d want=%d", prefix, wantBytes)
	}

	ext, ok := tree.Ext()
	if len(model) == 0 {
		if ok {
			t.Fatalf("empty tree should report Ext() as absent")
		}
		return
	}
	if !ok {
		t.Fatalf("non-empty tree should report Ext() as present")
	}
	if ext != wantBytes {
		t.Fatalf("Ext() mismatch: got=%d want=%d", ext, wantBytes)
	}
}

func runRandomExtensionSequence(t *testing.T, seed uint64, steps int) {
	t.Helper()
	r := rand.New(rand.NewSource(int64(seed)))
	tree := newExtTextTree(t, "ext:bytes")
	model := make([]string, 0, 64)

	for i := 0; i < steps; i++ {
		switch r.Intn(5) {
		case 0:
			pos := 0
			if len(model) > 0 {
				pos = r.Intn(len(model) + 1)
			}
			token := randomToken(r)
			var err error
			tree, err = tree.InsertAt(pos, FromString(token))
			if err != nil {
				t.Fatalf("InsertAt failed: %v", err)
			}
			model = append(model, "")
			copy(model[pos+1:], model[pos:])
			model[pos] = token
		case 1:
			if len(model) == 0 {
				continue
			}
			pos := r.Intn(len(model))
			var err error
			tree, err = tree.DeleteAt(pos)
			if err != nil {
				t.Fatalf("DeleteAt failed: %v", err)
			}
			model = append(model[:pos], model[pos+1:]...)
		case 2:
			if len(model) < 2 {
				continue
			}
			start := r.Intn(len(model) - 1)
			maxCount := len(model) - start
			count := r.Intn(maxCount) + 1
			var err error
			tree, err = tree.DeleteRange(start, count)
			if err != nil {
				t.Fatalf("DeleteRange failed: %v", err)
			}
			model = append(model[:start], model[start+count:]...)
		case 3:
			split := 0
			if len(model) > 0 {
				split = r.Intn(len(model) + 1)
			}
			left, right, err := tree.SplitAt(split)
			if err != nil {
				t.Fatalf("SplitAt failed: %v", err)
			}
			combined, err := left.Concat(right)
			if err != nil {
				t.Fatalf("Concat after split failed: %v", err)
			}
			tree = combined
		case 4:
			other := newExtTextTree(t, "ext:bytes")
			n := r.Intn(4)
			otherModel := make([]string, 0, n)
			for j := 0; j < n; j++ {
				token := randomToken(r)
				var err error
				other, err = other.InsertAt(other.Len(), FromString(token))
				if err != nil {
					t.Fatalf("other InsertAt failed: %v", err)
				}
				otherModel = append(otherModel, token)
			}
			var err error
			tree, err = tree.Concat(other)
			if err != nil {
				t.Fatalf("Concat failed: %v", err)
			}
			model = append(model, otherModel...)
		}
		assertTreeMatchesModelAndExtension(t, tree, model)
	}
}

func TestExtensionRandomizedProperty(t *testing.T) {
	seeds := []uint64{1, 2, 3, 7, 42, 99, 31337, 123456789}
	for _, seed := range seeds {
		t.Run("seed_"+strconv.FormatUint(seed, 10), func(t *testing.T) {
			runRandomExtensionSequence(t, seed, 80)
		})
	}
}

func FuzzExtensionRandomizedProperty(f *testing.F) {
	f.Add(uint64(1), uint8(32))
	f.Add(uint64(7), uint8(64))
	f.Add(uint64(42), uint8(96))
	f.Fuzz(func(t *testing.T, seed uint64, steps uint8) {
		runRandomExtensionSequence(t, seed, int(steps%120)+1)
	})
}
