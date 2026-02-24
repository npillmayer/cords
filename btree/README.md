# btree

`btree` is a B+ sum-tree backend for `cords`.

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

## Notes

- The package is still evolving and optimized for rope internals.
- Structural invariants are strict; internal inconsistencies are treated as
  implementation bugs and will panic.
