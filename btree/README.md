# btree

`btree` is an experimental B+ sum-tree backend for `cords`.

It is specialized for rope-like sequence editing (not a map/set container). The
tree stores leaf items `I` where each item provides a summary `S` via
`Summary() S`. Internal nodes aggregate summaries with a monoid.

## Purpose

- Provide a persistent (copy-on-write) tree for text editing workflows.
- Support positional operations by item index:
  - `InsertAt`
  - `DeleteAt`
  - `DeleteRange`
  - `SplitAt`
  - `Concat`
- Support summary-guided navigation with cursors and dimensions.

## Main API

- `New[I, S](cfg Config[S]) (*Tree[I, S], error)`
- `(*Tree).InsertAt(index int, items ...I) (*Tree[I, S], error)`
- `(*Tree).DeleteAt(index int) (*Tree[I, S], error)`
- `(*Tree).DeleteRange(index, count int) (*Tree[I, S], error)`
- `(*Tree).SplitAt(index int) (left, right *Tree[I, S], err error)`
- `(*Tree).Concat(other *Tree[I, S]) (*Tree[I, S], error)`
- `(*Tree).Len()`, `(*Tree).Summary()`, `(*Tree).Height()`
- `NewCursor(tree, dimension)` and `cursor.Seek(target)`

Useful built-in text types:

- `TextChunk`, `TextSummary`, `TextMonoid`
- `FromString(string) TextChunk`
- `ByteDimension`, `LineDimension`

## Example

```go
config := btree.Config[btree.TextSummary]{
	Monoid: btree.TextMonoid{},
}
t, _ := btree.New[btree.TextChunk, btree.TextSummary](config)

t, err := t.InsertAt(0,
	btree.FromString("hello "),
	btree.FromString("world\n"),
	btree.FromString("next line\n"),
)
if err != nil {
	log.Fatal(err)
}

// Delete "world\n" (item index 1).
t, err = t.DeleteAt(1)
if err != nil {
	log.Fatal(err)
}

// Split and join again.
left, right, err := t.SplitAt(1)
if err != nil {
	log.Fatal(err)
}
t, err = left.Concat(right)
if err != nil {
	log.Fatal(err)
}

// Seek by accumulated bytes.
cursor, _ := btree.NewCursor[btree.TextChunk, btree.TextSummary,
			uint64](t, btree.ByteDimension{})

idx, _, err := cursor.Seek(6)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("len=%d height=%d seek(6)->item=%d summary=%+v\n",
	t.Len(), t.Height(), idx, t.Summary())
```

## Notes

- The package is still evolving and optimized for rope internals.
- Structural invariants are strict; internal inconsistencies are treated as
  implementation bugs and will panic.
