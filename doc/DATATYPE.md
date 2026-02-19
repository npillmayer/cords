# Cords Data Type

## Chunks

Please refer to
[Zed's Chunks](https://github.com/zed-industries/zed/blob/main/crates/rope/src/chunk.rs)
for guidance.

### Source-backed findings (`doc/chunk.rs`)

Zed's implementation is richer than a "text + newline count" chunk. Key points:

- Chunk capacity is coupled to bitmap width:
  - `type Bitmap = u128` (tests may use `u16`)
  - `MAX_BASE = Bitmap::BITS`
  - `MIN_BASE = MAX_BASE / 2`
- `Chunk` stores:
  - `chars` bitmap (UTF-8 char starts),
  - `newlines` bitmap,
  - `tabs` bitmap,
  - `text: ArrayString<MAX_BASE>`.
- `Chunk::new` builds bitmaps from raw bytes in small fixed blocks, then
  initializes per-byte/per-char indexes.
- There is a zero-copy `ChunkSlice` view type that carries sliced/shifted bitmap
  state together with `&str`.
- Chunk and slice APIs provide extensive coordinate conversion primitives:
  - byte offset <-> line/column point,
  - byte offset <-> UTF-16 offset,
  - point <-> UTF-16 point,
  - row-range extraction and clipping.
- Clipping is grapheme-aware (`unicode_segmentation::GraphemeCursor`) and
  supports directional `Bias`.
- Tabs are exposed as an iterator yielding both byte offset and char offset.
- There are dedicated fast helpers for "n-th set bit" selection and extensive
  randomized tests validating all conversions.

### Evaluation against previous findings

Previous conclusions were directionally correct (fixed-size chunk + bitmaps),
but the actual source adds important details:

- `tabs` are first-class, not merely optional in current Zed code. This is suitable for a text-editor, but not a concern for our implementation.
- A `ChunkSlice` abstraction is central and should be considered explicitly.
- The chunk is not only for summary maintenance; it is a local coordinate engine
  (byte, UTF-16, line/column, grapheme clipping).
- Zed invests in correctness via property-style randomized testing; this is part
  of the design, not an afterthought.

### Implications for `cords`

1. Keep fixed-size inline chunk storage.
- This matches our cache/locality goals and keeps per-chunk work bounded.

2. Adopt bitmap-backed chunk indexes.
- Baseline: `chars`, `newlines`.
- Defer `chars_utf16`, as UTF-16 mapping is not currently required.
- Leave out `tabs`, as visual-column/tab metrics are not required.

3. Add a chunk-slice/view abstraction.
- Needed to avoid copying while splitting/concatenating chunks.
- Should carry shifted bitmap state, like Zed's `ChunkSlice`.

4. Separate concerns cleanly.
- Tree summary routes to the right chunk (`O(log n)`).
- Chunk does dense local coordinate math (`O(1)`/small bounded loops).
- Status: implemented. `chunk` now owns `Summary` (`bytes/chars/lines`),
  `Monoid`, and dimension types (`ByteDimension`, `CharDimension`,
  `LineDimension`). `btree` remains generic and consumes these via interfaces
  (`SummaryMonoid`, `Dimension`) without text-specific logic.

5. Mirror Zed's testing style.
- Add randomized round-trip tests for all coordinate transforms and clipping.

### Go-specific design note

Go has no native `u128`. Therefore we use `uint64` bitmaps and set chunk capacity to 64 bytes.
For an initial implementation, `uint64` chunks are simpler and still align well
with the architecture demonstrated by Zed.

### Ingestion reminder

When building chunks from file I/O, chunk boundaries must align to UTF-8 rune
boundaries. A splitter must not cut inside a multi-byte rune. `chunk.NewBytes`
validates input and will reject invalid UTF-8 slices, but the read/chunking
pipeline should preserve rune boundaries proactively.

## Implementation Plan (Zed-style chunks for `cords`)

1. Define chunk core types in a new package (e.g. `chunk/`).
- `type Bitmap = uint64`
- constants: `MaxBase=64`, `MinBase=32`
- `type Chunk struct { chars, newlines Bitmap; text [MaxBase]byte; n uint8 }`

2. Add construction and immutable editing primitives.
- `New(string) (Chunk, error)` (validate UTF-8, build bitmaps)
- `Len()`, `String()`, `Bytes()`
- `Slice(range) ChunkSlice`
- `SplitAt(i) (ChunkSlice, ChunkSlice)`
- `Append(ChunkSlice) (Chunk, bool)` where `bool` signals fit/overflow

3. Add `ChunkSlice` zero-copy view (like Zed).
- fields: shifted bitmaps + `[]byte`/`string` view
- operations: `Len`, `IsCharBoundary`, `Slice`, `SplitAt`
- keep it cheap and allocation-free

4. Implement coordinate conversions (first milestone).
- `Offset -> Point(row,col)` (bytes)
- `Point -> Offset`
- defer UTF-16 conversion APIs
- skip grapheme-clipping initially; add later

5. Define summary for tree integration.
- `ChunkSummary` with at least: `bytes`, `chars`, `lines`
- optional later: `firstLineChars`, `lastLineChars`, `longestRow`
- implement monoid for `btree`

6. Add adapters to current code paths.
- btree: make chunk item satisfy `Summary()`
- base package (`Leaf` interface): wrapper implementing
  `Weight/String/Substring/Split` using chunk operations

7. Testing strategy (important).
- deterministic edge tests: ASCII, multibyte UTF-8, 4-byte runes, newline boundaries
- randomized/property tests: split/slice round-trips and all coordinate conversion round-trips
- invariants: bitmap consistency vs actual bytes

8. Benchmarks and rollout.
- compare old `StringLeaf`/`TextChunk` vs new `Chunk` for insert/split/report workloads
- once stable, switch btree text item default to chunk-backed type
