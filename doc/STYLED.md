# styled package scan

Date: 2026-02-20

## Scope

This scan covers package `styled` and sub-packages:

- `styled`
- `styled/formatter`
- `styled/inline`
- `styled/itemized`

## Build status (current)

Command:

```sh
GOCACHE=/tmp/go-build go test ./styled/...
```

Result: **build fails**.

Primary compiler errors are API incompatibilities with current `cords`:

- `undefined: cords.Leaf` (`styled/builder.go:44`, `styled/styles.go:250`, `styled/styles.go:269`, `styled/paragraph.go:98`)
- missing `Builder.Append` method (`styled/builder.go:51`)
- missing `Cord.EachLeaf` method (`styled/styles.go:68`, `styled/styles.go:137`, `styled/paragraph.go:98`)
- invalid type assertion because `Cord.Index` returns `chunk.Chunk` now (`styled/styles.go:55`)

Because `styled` root fails to compile, `styled/formatter`, `styled/inline`, and `styled/itemized` also fail transitively.

## Documentation findings

### Top-level docs

- `styled/doc.go` says only “Work in progress” and has no API guidance for current architecture (`styled/doc.go:2-7`).
- `styled/ReadMe.md` is only a warning and does not document intended stable semantics (`styled/ReadMe.md:1-3`).

### Sub-package docs

- `styled/formatter/doc.go` has substantial conceptual material and good intent, but still marks package/API as unstable (`styled/formatter/doc.go:70-74`).
- `styled/inline/doc.go` and `styled/itemized/doc.go` are mostly license boilerplate with little practical usage/migration detail (`styled/inline/doc.go:1-7`, `styled/itemized/doc.go:1-3`).
- No document in `styled/` explains migration from legacy leaf-based ropes to current chunk/btree cords.

## Code findings

### 1) Core migration blockers in `styled/`

- `TextBuilder` still depends on old leaf-based append:
  - signature `Append(leaf cords.Leaf, ...)` (`styled/builder.go:44`)
  - call to removed `b.cordBuilder.Append(leaf)` (`styled/builder.go:51`)
- style-run storage uses `type runs cords.Cord` plus synthetic `styleLeaf` implementing old leaf interface (`styled/styles.go:152`, `styled/styles.go:224-269`).
- iteration depends on removed `EachLeaf` API (`styled/styles.go:68`, `styled/styles.go:137`, `styled/paragraph.go:98`).
- `StyleAt` logic assumes index returns interface-like leaf type and performs incompatible assertion (`styled/styles.go:51-56`).

### 2) Additional correctness issues (independent of migration)

- `TextBuilder.Text()` has a value receiver, so `b.done = true` mutates only a copy (`styled/builder.go:29-31`).
- `inline.Style.Equals` always returns `false` (`styled/inline/styles.go:88-90`).
- `itemized.Iterator.Style()` panics on substring error (`styled/itemized/items.go:109-112`) instead of propagating error.
- `ConsoleFixedWidth.Postamble()` writes `Preamble` bytes, not `Postamble` (`styled/formatter/console.go:186-188`).
- `HTML.Postamble()` writes `"<pre>"` instead of closing `"</pre>"` (`styled/formatter/html.go:118-123`).
- `HTMLStyle.Add()` recursively calls itself (`styled/formatter/html.go:209-210`).
- `TestVTE` is a hard-coded failing TODO (`styled/formatter/fmt_test.go:66`).

### 3) Areas that still make conceptual sense

- Separation of concerns is still clear in design:
  - `styled.Text` holds raw text plus style runs (`styled/styles.go:11-15`)
  - `styled.Paragraph` adds bidi/layout preparation (`styled/paragraph.go:10-31`)
  - `formatter` models output-target abstraction (`styled/formatter/format.go:59-69`)
  - `itemized` provides style-run iteration API (`styled/itemized/items.go:41-48`)
- The package intent (immutable text + overlays of style runs + bidi-aware formatting) remains coherent.

## Summary assessment

- **Intent quality:** high.
- **Current implementation compatibility with new cords:** low.
- **Documentation quality for current state:** low-to-medium.
- **Technical debt concentration:** mostly in `styled` root API coupling to removed leaf primitives.
