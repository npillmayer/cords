# Rune-Aware Positioning for `cords`

## Status

The rune-aware positioning scheme is now implemented in the base `cords` package.

Implemented pieces:

- `Pos` (opaque value type with rune + byte coordinate)
- position conversion APIs (`PosFromByte`, `ByteOffset`, `PosStart`, `PosEnd`)
- strict position validation (`ErrIllegalPosition`)
- `CharCursor` (`SeekPos`, `SeekRunes`, `Next`, `Prev`)
- rune convenience wrappers (`ReportRunes`, `SplitRunes`)
- btree helper `PrefixSummary(...)` for efficient prefix summary lookup

## Design Summary

The API keeps byte-based operations as the low-level canonical layer and adds a rune-based abstraction on top.

- byte APIs remain unchanged (`Split`, `Report`, `Index`, etc.)
- rune APIs convert via validated `Pos` and route back to byte operations
- semantics are rune-based (Unicode scalar values), not grapheme-based

## Types

```go
// Opaque to external callers (unexported fields).
type Pos struct {
    runes   uint64
    bytepos uint64
}

type CharCursor struct {
    cord    Cord
    pos     Pos
    byteOff uint64
}
```

## Pos Invariant

For any `Pos` created from a cord snapshot:

- `runes` and `bytepos` refer to the same boundary,
- `runes` is the rune offset at that boundary,
- `bytepos` is the byte offset at that boundary.

No public API constructs partial/inconsistent `Pos` values.

## Public API (Current)

```go
// Position endpoints
func (c Cord) PosStart() Pos
func (c Cord) PosEnd() Pos

// Conversion bridges
func (c Cord) PosFromByte(b uint64) (Pos, error)
func (c Cord) ByteOffset(p Pos) (uint64, error)

// Rune cursor
func (c Cord) NewCharCursor() (*CharCursor, error)
func (cc *CharCursor) Pos() Pos
func (cc *CharCursor) ByteOffset() uint64
func (cc *CharCursor) SeekPos(p Pos) error
func (cc *CharCursor) SeekRunes(n uint64) error
func (cc *CharCursor) Next() (r rune, ok bool)
func (cc *CharCursor) Prev() (r rune, ok bool)

// Rune wrappers
func SplitRunes(cord Cord, p Pos) (Cord, Cord, error)
func (c Cord) ReportRunes(start Pos, n uint64) (string, error)
```

Internal (intentionally non-public for now):

- `posFromRunes(...)`

## Validation Rule for Pos Consumers

All `Pos`-consuming APIs follow this validation pattern:

1. Check `pos.bytepos <= cord.Len()`.
2. Resolve canonical position from byte offset via `PosFromByte(pos.bytepos)`.
3. Compare resolved rune offset with `pos.runes`.
4. Mismatch => `ErrIllegalPosition`.

This detects many cross-cord / stale-position mistakes while keeping conversion deterministic.

## Errors

- `ErrIndexOutOfBounds`: offset/span exceeds cord bounds.
- `ErrIllegalPosition`: byte/rune coordinate mismatch for target cord.

## B-Tree / Chunk Integration

- Rune counts already existed as `chunk.Summary.Chars`.
- Rune routing dimension already existed as `chunk.CharDimension`.
- Added `Tree.PrefixSummary(itemIndex)` to compute prefix summaries without split-copy.
- Local chunk conversion uses UTF-8 boundary bitmap information from `chunk.Chunk`.

## Complexity Notes

- byte<->rune seek through tree: `O(log n)`
- local conversion inside chunk: `O(chunk_size)` worst case (bounded by `chunk.MaxBase`)

## Test Coverage Added

- byte->pos->byte roundtrip
- non-boundary byte rejection
- mismatch/foreign-pos rejection
- cross-chunk multibyte conversion
- cursor forward/backward traversal
- cursor seek correctness
- rune wrapper behavior and error handling

## Remaining / Open

- Optional stamp-based identity checks (stronger cross-cord/stale detection)
- Public `PosFromRunes` decision
- Grapheme-level cursor/positioning (future layer, separate semantics)
