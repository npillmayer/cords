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
5. Regression tests in `styled/styles_test.go` cover:
   - basic styling,
   - merge of adjacent equal styles,
   - empty-span no-op,
   - whole-text styling,
   - `StyleAt` behavior and bounds.
6. `TextBuilder.Append` has been migrated to chunk-based append (`AppendChunk`).

## Missing pieces for synchronization

The central migration gap is still present: `Text` has no complete editing API that updates both raw text and style runs together.

### Missing run operations (needed as primitives)

1. split runs at byte position (`Runs.SplitAt`-like primitive)
2. concat runs with boundary merge (`Runs.Concat`-like primitive)
3. extract subsection of runs (`Runs.Section`-like primitive)
4. adjust runs for inserted/deleted text ranges (insert/delete helpers at run level)

Without these, synchronized implementations of text-edit operations are not practical.

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

1. Introduce run-level structural operations (`SplitAt`, `Concat`, `Section`, range adjust helpers).
2. Implement text-edit APIs on `Text` that always update raw cord and runs atomically.
3. Re-enable style-run iteration/reporting APIs (`StyleRuns`, `EachStyleRun`) on top of btree runs.
4. Rework `ParagraphFromText` and `WrapAt` to use run-level split/concat primitives.
5. Remove obsolete `styleLeaf` legacy scaffolding after replacement APIs are in place.

## Summary

`styled` has successfully migrated core style storage and lookup to btree, and styling operations now work with tests. The remaining refactoring work is primarily about synchronized text editing and paragraph operations, which require dedicated run-structure primitives first.

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

## Implementation Plan: First Primitive `Runs.SplitAt(pos)`

Goal: split a `Runs` value by byte position `pos` into `(left, right)` such that:

1. `left` covers `[0,pos)`,
2. `right` covers `[pos,total)`,
3. both outputs satisfy the run invariant,
4. `left + right` reconstructs the original run stream.

Step plan:

1. Define API and contract:
   - `func (runs Runs) SplitAt(pos uint64) (Runs, Runs, error)`.
   - bounds: `0 <= pos <= totalRunLength`.
   - edge cases: `pos=0`, `pos=total`, empty runs.
2. Add tests first in `styled/styles_test.go`:
   - split at 0 and end,
   - split exactly at run boundary,
   - split inside one run (run must be divided),
   - split across multi-run structures,
   - reconstruction check via planned `Concat` seam helper or test-local run concatenation,
   - invariant checks for both outputs.
3. Implement length/bounds pre-check using run summary length.
4. Reuse cursor seek (`StyleDimension`) to locate the run containing `pos-1` or seam boundary logic for exact boundary.
5. Build left/right replacement around seam:
   - if split is mid-run, create two fragments of that run;
   - if split is at boundary, no fragment split needed.
6. Construct result trees with minimal edits:
   - either via tree-level split + seam patch, or via range replacement using current helpers.
7. Run local seam-repair on both outputs near their boundary nodes:
   - left tail and right head only.
8. Assert/verify invariants in debug path:
   - no zero-length runs,
   - no adjacent equal-style runs,
   - length conservation (`len(left)+len(right)==len(original)`).
9. Keep helper abstractions reusable for next operators:
   - `repairAround(tree, leftIndex)` or equivalent seam merge helper.
10. After `SplitAt` lands and tests pass, implement `Runs.Concat` next using the same seam-repair helper.
