# styled package scan

Date: 2026-02-25

## Scope

This scan covers package `styled` only.

Explicitly out of scope for this document: `styled/formatter`, `styled/inline`, `styled/itemized`.

## Build/test status (current)

Command:

```sh
go test ./styled -count=1
```

Result: **passes**.

## Status vs target architecture

Target architecture:

- `Text` stores raw text as a `cords.Cord`.
- Styling metadata is stored separately as run segments (`Runs`) in a btree.
- Text and runs must stay synchronized across text manipulations.

Current implementation status:

- `Text` already has the intended data split:
  - raw: `text cords.Cord`
  - styles: `runs Runs`
- `Runs` is now btree-backed:
  - `btree.Tree[Run, Summary, btree.NO_EXT]`
- style lookup (`StyleAt`) uses btree cursor seek over run-length summaries.
- style application (`Text.Style` / `Runs.Style`) rewrites only the affected run window and merges adjacent equal styles.

## What is working

1. Core run model is migrated to btree.
2. Initial styling (`initialStyle`) creates a full run partition over text length.
3. Incremental restyling (`Runs.Style`) replaces affected runs and keeps invariants.
4. `StyleAt` returns `(style, offsetWithinRun)` and is test-covered.
5. Primitive run operators are implemented in `styled/run_ops.go`:
   - `Runs.SplitAt(pos)`
   - `Runs.Concat(other)`
   - `Runs.Section(from, to)`
   - `Runs.DeleteRange(from, to)`
   - `Runs.InsertAt(pos, n, sty)`
6. Regression tests in `styled/styles_test.go` cover:
   - basic styling,
   - merge of adjacent equal styles,
   - empty-span no-op,
   - whole-text styling,
   - `StyleAt` behavior and bounds.
7. Dedicated operator tests in `styled/run_ops_test.go` cover:
   - split edge/boundary/in-run/bounds behavior,
   - concat seam merge + non-merge + empty cases,
   - section whole/empty/range/bounds behavior,
   - delete-range no-op/bounds/range/edge/full-delete behavior,
   - insert no-op/bounds/seam-merge/in-run behavior.
8. `TextBuilder.Append` has been migrated to chunk-based append (`AppendChunk`).

## Missing pieces for synchronization

The central migration gap is still present: `Text` has no complete editing API that updates both raw text and style runs together.

### Remaining run-operation gaps

1. optional normalization/repair helper abstraction shared by all run mutations (currently implicit in operator logic)
2. explicit policy documentation for inserted-run style semantics (currently implemented as caller-provided style)

These are refinement tasks; the main remaining blocker is wiring synchronized edit operations at `Text`/`Paragraph` level.

### Missing/disabled Text & Paragraph APIs

1. `Text.Section` is commented out.
2. `Text.StyleRuns` is commented out.
3. `Text.EachStyleRun` is commented out.
4. `Paragraph.EachStyleRun` is stubbed and returns `nil`.
5. `Paragraph.StyleRuns` is stubbed and returns `nil`.
6. `ParagraphFromText` cannot construct sub-paragraphs yet (`Section` path disabled).
7. `Paragraph.WrapAt` currently splits raw text only; style-run split/sync is not implemented.

## Consistency risks in current state

1. Raw-text operations inside `Paragraph` can diverge from style runs because run-sync logic is not wired (`WrapAt` path).
2. Legacy code artifacts from pre-btree implementation (`styleLeaf` and large commented blocks) increase maintenance noise and can obscure current behavior.

## Proposed next batches

1. Implement synchronized text-edit APIs on `Text` (insert/delete-range) that update raw cord and runs atomically.
2. Implement `Text.Section` using `Runs.Section`.
3. Re-enable style-run iteration/reporting APIs (`StyleRuns`, `EachStyleRun`) on top of btree runs.
4. Rework `ParagraphFromText` and `WrapAt` to use `Runs.Section`/split/concat and maintain style sync.
5. Remove obsolete `styleLeaf` legacy scaffolding after replacement APIs are in place.

## Summary

`styled` has successfully migrated core style storage, lookup, and the basic run operator set (`SplitAt`, `Concat`, `Section`, `DeleteRange`, `InsertAt`) to btree, with dedicated tests. The remaining refactoring work is primarily synchronized `Text`/`Paragraph` editing and API re-enablement on top of these primitives.

## Decision Record: Run-Coalescing Strategy

Invariant to preserve after every `Runs` operation:

1. no zero-length runs,
2. no adjacent runs with equal `Style`,
3. total run length equals owning text length (when attached to a `Text`).

Two implementation styles were considered:

1. pre-coalesce (analyze/adjust replacement scope before writing), as in current `Runs.Style(...)`;
2. post-repair (perform operation first, then coalesce).

Trade-off assessment:

1. pre-coalesce can minimize writes, but pushes complex seam logic into every operator implementation;
2. full-tree post-repair is cleaner conceptually but too expensive to run after each operation;
3. local post-repair around changed seams keeps APIs clean and has near-constant repair work per operation.

Decision:

Adopt **local seam-repair after mutation** as the default strategy for new `Runs` primitives.  
Do not do full-tree coalescing by default.

## Progress update on run primitives

Completed since previous scan:

1. `Runs.SplitAt(pos)` implemented and test-covered.
2. `Runs.Concat(other)` implemented with seam coalescing and test-covered.
3. `Runs.Section(from, to)` implemented via `SplitAt` composition and test-covered.
4. `Runs.DeleteRange(from, to)` implemented and test-covered.
5. `Runs.InsertAt(pos, n, sty)` implemented and test-covered.
6. Run-operation tests moved into dedicated file `styled/run_ops_test.go`.

Immediate next integration target:

1. synchronized raw-text edit operations on `Text` (insert/delete-range) using the run primitives.
