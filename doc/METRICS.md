# Metrics and Summary Extensibility (`cords` / `cordext` / `btree`)

Status snapshot: 2026-02-28

This document describes what is implemented today.

---

## 1. Implemented Extension Primitive (in `btree`)

The generic sum-tree already supports extension summaries.

- `btree.Config[I,S,E]` includes optional `Extension SumExtension[I,S,E]`.
- `SumExtension` defines:
  - `MagicID() string`
  - `Zero() E`
  - `FromItem(I,S) E`
  - `Add(E,E) E`
- Extension values are aggregated in nodes alongside base summaries.
- Cross-tree operations check extension compatibility via `MagicID`.

### Cursor / Dimension Model

Dimension-based seeking is implemented for both base summaries and extension summaries:

- `Cursor` uses `Dimension[S,K]` (summary-driven).
- `ExtCursor` uses `Dimension[E,K]` (extension-driven).
- Both support `Seek` and `SeekItem`.

`Dimension` is:

```go
type Dimension[S any, K any] interface {
    Zero() K
    Add(acc K, summary S) K
    Compare(acc K, target K) int
}
```

---

## 2. Implemented Extension Host for Text (in `cordext`)

`cordext` provides extension-enabled cords:

- `CordEx[E]` stores text in `btree.Tree[chunk.Chunk, chunk.Summary, E]`.
- `FromStringWithExtension(s, ext)` constructs extension-enabled text.
- `FromStringNoExt(s)` constructs `CordEx[btree.NO_EXT]`.
- `Ext()` returns aggregated extension value.
- `PrefixExt(itemIndex)` returns extension prefix aggregate.
- `NewExtCursor(cord, dim)` provides extension-driven seek.

`cordext` extension contract:

```go
type TextSegmentExtension[E any] interface {
    MagicID() string
    Zero() E
    FromSegment(TextSegment) E
    Add(E, E) E
}
```

This is adapted internally to `btree.SumExtension`.

---

## 3. Root `cords` Package Status

The root `cords.Cord` type is still intentionally no-extension:

- `Cord` uses `btree.Tree[chunk.Chunk, chunk.Summary, btree.NO_EXT]`.
- Internally, root package operations are bridged to `cordext` no-extension paths.

So:

- Extension support is available today through `cordext`.
- Root `cords` does not yet expose runtime extension/schema configuration.

---

## 4. Metrics Package Status (`metrics`)

The package is now concrete and analyzer-oriented (legacy generic framework removed).

Implemented analyzers:

1. `Words()` materializer over `cords.Cord`
   - API: `Apply(text, i, j) -> (WordsValue, materialized Cord, error)`

2. Paragraph discovery over `cordext.CordEx[btree.NO_EXT]`
   - `FindParagraphs(text, policy) []ParagraphSpan`
   - `ParagraphAt(text, pos, policy) (ParagraphSpan, error)`
   - `ParagraphsInRange(text, from, to, policy) ([]ParagraphSpan, error)`
   - policy supports:
     - line-break delimiters (`\n`, `\r\n`)
     - blank-line delimiters
     - keep-empty behavior

---

## 5. What Is No Longer Accurate

Older statements proposing sidecar-only metrics as the main path are outdated.

Today:

- `btree` already has native extension aggregation.
- `cordext` already exposes an extension-enabled cord API.
- `metrics` already contains concrete analyzers and paragraph APIs.

---

## 6. Practical Guidance

If you need custom metric/summary behavior now:

1. Use `cordext.CordEx[E]` with a `TextSegmentExtension[E]`.
2. Use `ExtCursor`/`PrefixExt` for extension-driven queries.
3. Keep root `cords.Cord` as the stable, no-extension client API unless extension data is required.

If you only need text analysis (not structural extension summaries), add focused analyzers to `metrics` on top of `cordext` segment iteration.

---

## 7. Paragraph Discovery Optimization Notes (2026-02-28)

Current `metrics.FindParagraphs` behavior:

- implementation scans text segments and bytes with a small state machine,
- detects `\n` and `\r\n` (`\r` alone is intentionally not a break),
- derives paragraph separators and finally `[from,to)` paragraph spans.

This is correct, but does not yet exploit tree summary routing.

### Opportunity

A paragraph break must be related to line-break structure. The tree already stores
line counts in chunk summaries (`Summary.Lines`), so we can avoid scanning chunks
that cannot contain line breaks.

### Evaluated Strategy: Cursor by Line Rank

Use a cursor with `LineDimension` to iterate line-break ranks (1..L), then map each
rank to chunk/local position and classify whether it is a paragraph separator.

Pros:

1. avoids full-byte scans in sparse-line-break texts,
2. uses existing summary + dimension primitives.

Cons:

1. naive repeated `SeekItem` is `O(L log C)` (`L` line breaks, `C` chunks),
2. still needs local context handling for `\r\n`, blank-line policy, and boundary rules,
3. CRLF (`\r\n`) adjacency still requires local byte context handling around
   chunk boundaries.

### Recommended Near-Term Optimization

Prefer a chunk-summary-driven scan before introducing rank-seek iteration:

1. iterate segments once,
2. skip chunks with `LineCount()==0`,
3. for relevant chunks, enumerate newline offsets from chunk newline bitmaps,
4. keep minimal cross-chunk state for CRLF handling.

This keeps complexity low while moving runtime closer to `O(C + L)` for common
policies.
