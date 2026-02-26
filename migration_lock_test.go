package cords

import (
	"math/rand"
	"testing"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/chunk"
	"github.com/npillmayer/cords/cordext"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestBehaviorLockDeterministicAgainstCordextNoExt(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	base := "Hello\n🙂 World\nß!"
	c := FromString(base)
	x := toCordext(c)
	assertCordParity(t, c, x)

	extra := " ++"
	c = Concat(c, FromString(extra))
	x, err := x.Concat(toCordext(FromString(extra)))
	if err != nil {
		t.Fatalf("cordext concat failed: %v", err)
	}
	assertCordParity(t, c, x)

	b := runeBoundaries(c.String())
	insertAt := b[3]
	if c, err = Insert(c, FromString("Ω"), insertAt); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if x, err = x.Insert(toCordext(FromString("Ω")), insertAt); err != nil {
		t.Fatalf("cordext insert failed: %v", err)
	}
	assertCordParity(t, c, x)

	b = runeBoundaries(c.String())
	from, to := b[2], b[6]
	cutLen := to - from
	var cut Cord
	var cutX cordext.CordEx[btree.NO_EXT]
	if c, cut, err = Cut(c, from, cutLen); err != nil {
		t.Fatalf("cut failed: %v", err)
	}
	if x, cutX, err = x.Cut(from, cutLen); err != nil {
		t.Fatalf("cordext cut failed: %v", err)
	}
	if cut.String() != cutX.String() {
		t.Fatalf("cut payload mismatch: root=%q ext=%q", cut.String(), cutX.String())
	}
	assertCordParity(t, c, x)

	b = runeBoundaries(c.String())
	splitAt := b[len(b)/2]
	l, r, err := Split(c, splitAt)
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}
	lx, rx, err := x.Split(splitAt)
	if err != nil {
		t.Fatalf("cordext split failed: %v", err)
	}
	if l.String() != lx.String() || r.String() != rx.String() {
		t.Fatalf("split mismatch: root=(%q,%q) ext=(%q,%q)", l.String(), r.String(), lx.String(), rx.String())
	}
	c = Concat(l, r)
	x, err = lx.Concat(rx)
	if err != nil {
		t.Fatalf("cordext concat failed: %v", err)
	}
	assertCordParity(t, c, x)

	b = runeBoundaries(c.String())
	subFrom, subTo := b[1], b[len(b)-1]
	subLen := subTo - subFrom
	sub, err := Substr(c, subFrom, subLen)
	if err != nil {
		t.Fatalf("substr failed: %v", err)
	}
	subX, err := x.Substr(subFrom, subLen)
	if err != nil {
		t.Fatalf("cordext substr failed: %v", err)
	}
	if sub.String() != subX.String() {
		t.Fatalf("substr mismatch: root=%q ext=%q", sub.String(), subX.String())
	}

	report, err := c.Report(0, c.Len())
	if err != nil {
		t.Fatalf("report failed: %v", err)
	}
	reportX, err := x.Report(0, x.Len())
	if err != nil {
		t.Fatalf("cordext report failed: %v", err)
	}
	if report != reportX {
		t.Fatalf("report mismatch: root=%q ext=%q", report, reportX)
	}
}

func TestBehaviorLockRandomizedAgainstCordextNoExt(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()

	rnd := rand.New(rand.NewSource(0xC0D5))
	c := FromString("alpha\nbeta🙂gamma")
	x := toCordext(c)
	assertCordParity(t, c, x)

	pool := []string{"a", "Z", "ß", "🙂", "\n", "xy"}
	for step := 0; step < 400; step++ {
		switch rnd.Intn(5) {
		case 0: // insert
			ins := pool[rnd.Intn(len(pool))]
			pos := uint64(rnd.Intn(int(c.Len() + 2)))
			var (
				next  Cord
				nextX cordext.CordEx[btree.NO_EXT]
				err   error
				errX  error
			)
			next, err = Insert(c, FromString(ins), pos)
			nextX, errX = x.Insert(toCordext(FromString(ins)), pos)
			requireErrorParity(t, err, errX, "insert", step)
			if err == nil {
				c, x = next, nextX
			}
		case 1: // cut
			from := uint64(rnd.Intn(int(c.Len() + 2)))
			l := uint64(rnd.Intn(int(c.Len() + 2)))
			var (
				next  Cord
				cut   Cord
				nextX cordext.CordEx[btree.NO_EXT]
				cutX  cordext.CordEx[btree.NO_EXT]
				err   error
				errX  error
			)
			next, cut, err = Cut(c, from, l)
			nextX, cutX, errX = x.Cut(from, l)
			requireErrorParity(t, err, errX, "cut", step)
			if err == nil {
				if cut.String() != cutX.String() {
					t.Fatalf("step %d cut payload mismatch: root=%q ext=%q", step, cut.String(), cutX.String())
				}
				c, x = next, nextX
			}
		case 2: // split + join
			pos := uint64(rnd.Intn(int(c.Len() + 2)))
			l, r, err := Split(c, pos)
			lx, rx, errX := x.Split(pos)
			requireErrorParity(t, err, errX, "split", step)
			if err == nil {
				if l.String() != lx.String() || r.String() != rx.String() {
					t.Fatalf("step %d split mismatch", step)
				}
				c = Concat(l, r)
				x, errX = lx.Concat(rx)
				if errX != nil {
					t.Fatalf("step %d cordext concat failed: %v", step, errX)
				}
			}
		case 3: // substr check
			from := uint64(rnd.Intn(int(c.Len() + 2)))
			l := uint64(rnd.Intn(int(c.Len() + 2)))
			sub, err := Substr(c, from, l)
			subX, errX := x.Substr(from, l)
			requireErrorParity(t, err, errX, "substr", step)
			if err == nil && sub.String() != subX.String() {
				t.Fatalf("step %d substr mismatch: root=%q ext=%q", step, sub.String(), subX.String())
			}
		case 4: // report + index check
			from := uint64(rnd.Intn(int(c.Len() + 2)))
			l := uint64(rnd.Intn(int(c.Len() + 2)))
			rep, err := c.Report(from, l)
			repX, errX := x.Report(from, l)
			requireErrorParity(t, err, errX, "report", step)
			if err == nil && rep != repX {
				t.Fatalf("step %d report mismatch: root=%q ext=%q", step, rep, repX)
			}
			pos := uint64(rnd.Intn(int(c.Len() + 2)))
			ch, off, err := c.Index(pos)
			chX, offX, errX := x.Index(pos)
			requireErrorParity(t, err, errX, "index", step)
			if err == nil {
				if off != offX {
					t.Fatalf("step %d index offset mismatch: root=%d ext=%d", step, off, offX)
				}
				if byteAtChunk(ch, off) != byteAtChunk(chX, offX) {
					t.Fatalf("step %d index byte mismatch", step)
				}
			}
		}
		assertCordParity(t, c, x)
	}
}

func assertCordParity(t *testing.T, c Cord, x cordext.CordEx[btree.NO_EXT]) {
	t.Helper()
	if c.String() != x.String() {
		t.Fatalf("string mismatch: root=%q ext=%q", c.String(), x.String())
	}
	if c.Len() != x.Len() {
		t.Fatalf("length mismatch: root=%d ext=%d", c.Len(), x.Len())
	}
	if c.Summary() != x.Summary() {
		t.Fatalf("summary mismatch: root=%+v ext=%+v", c.Summary(), x.Summary())
	}
}

func requireErrorParity(t *testing.T, err error, errX error, op string, step int) {
	t.Helper()
	if (err == nil) != (errX == nil) {
		t.Fatalf("step %d %s error parity mismatch: root=%v ext=%v", step, op, err, errX)
	}
}

func byteAtChunk(c chunk.Chunk, off uint64) byte {
	return c.AsSlice().Bytes()[int(off)]
}

func runeBoundaries(s string) []uint64 {
	b := make([]uint64, 0, len(s)+1)
	for i := range s {
		b = append(b, uint64(i))
	}
	b = append(b, uint64(len(s)))
	return b
}
