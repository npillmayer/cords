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

- `Text` stores raw text as a `cordext.CordEx[btree.NO_EXT]`.
- Styling metadata is stored separately as run segments (`Runs`) in a btree.
- Text and runs must stay synchronized across text manipulations.

Current implementation status:

- `Text` already has the intended data split:
  - raw: `text cordext.CordEx[btree.NO_EXT]`
  - styles: `runs Runs`
- `Runs` is now btree-backed:
  - `btree.Tree[Run, Summary, btree.NO_EXT]`
- run-operator API error semantics are now package-local (`styled.ErrIndexOutOfBounds`, `styled.ErrIllegalArguments`, `styled.ErrVoidRuns`) instead of forwarding `cords` errors.
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
6. Run operations have been reworked to use internal pipeline helpers (`styled/monad.go`) for composing operation steps.
7. Regression tests in `styled/styles_test.go` cover:
   - basic styling,
   - merge of adjacent equal styles,
   - empty-span no-op,
   - whole-text styling,
   - `StyleAt` behavior and bounds.
8. Dedicated operator tests in `styled/run_ops_test.go` cover:
   - split edge/boundary/in-run/bounds behavior,
   - concat seam merge + non-merge + empty cases,
   - section whole/empty/range/bounds behavior,
   - delete-range no-op/bounds/range/edge/full-delete behavior,
   - insert no-op/bounds/seam-merge/in-run behavior.
9. Synchronized text-edit operations in `styled/text_ops.go` are implemented:
   - `Text.DeleteRange(from, to)` updates raw text and run metadata consistently,
   - `Text.InsertAt(pos, insertion, sty)` handles styled and unstyled insertions,
   - `Text.Concat(other)` concatenates raw text and run trees, including seam merge.
10. Dedicated tests in `styled/text_ops_test.go` cover `Text.DeleteRange`, `Text.InsertAt`, and `Text.Concat` including bounds/no-op/error paths and run/text synchronization invariants.
11. `TextBuilder.Append` has been migrated to chunk-based append (`AppendChunk`).

## Missing pieces for synchronization

The central migration gap has narrowed: core synchronized edits on `Text` are now in place (`DeleteRange`, `InsertAt`, `Concat`), but sectioning/iteration and paragraph-level synchronization are still incomplete.

### Remaining run-operation gaps

1. optional normalization/repair helper abstraction shared by all run mutations (currently implicit in operator logic)
2. explicit policy documentation for inserted-run style semantics (currently implemented as caller-provided style)
3. clarify and document error-contract mapping for each run op (`ErrIllegalArguments` vs `ErrIndexOutOfBounds`).

These are refinement tasks; the main remaining blockers are section/iteration APIs and paragraph-level synchronization.

### Missing/disabled Text & Paragraph APIs

1. `Text.StyleRuns` is commented out.
2. `Text.EachStyleRun` is commented out.
3. `Paragraph.EachStyleRun` is stubbed and returns `nil`.
4. `Paragraph.StyleRuns` is stubbed and returns `nil`.
5. `ParagraphFromText` cannot construct sub-paragraphs yet (`Section` path disabled).
6. `Paragraph.WrapAt` currently splits raw text only; style-run split/sync is not implemented.

## Consistency risks in current state

1. Raw-text operations inside `Paragraph` can diverge from style runs because run-sync logic is not wired (`WrapAt` path).
2. `Text.Concat` currently materializes plain runs when one side is styled and the other side is unstyled; this is correct for synchronization but should be documented as canonical semantics for mixed styled/unstyled concatenation.
3. Legacy code artifacts from pre-btree implementation (`styleLeaf` and large commented blocks) increase maintenance noise and can obscure current behavior.
4. Pipeline-based error propagation (`styled/monad.go`) is now central to run ops; negative/error-path coverage should be expanded to lock behavior.

## Proposed next batches

1. Re-enable style-run iteration/reporting APIs (`StyleRuns`, `EachStyleRun`) on top of btree runs.
2. Rework `ParagraphFromText` and `WrapAt` to use run primitives (`Section`/`SplitAt`/`Concat`) and maintain style sync.
3. Define and document canonical semantics for unstyled runs (`ErrVoidRuns`) across all `Text` operations.
4. Remove obsolete `styleLeaf` legacy scaffolding after replacement APIs are in place.

## Summary

`styled` has successfully migrated core style storage, lookup, run operators, and synchronized `Text` editing (`DeleteRange`, `InsertAt`, `Concat`, `Section`) to the btree-based model, with dedicated tests. Remaining refactoring work is now centered on style-run iteration/reporting and paragraph-level style synchronization.

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
7. Run ops were reworked to use internal pipeline helpers and package-local error values.
8. Synchronized `Text` edit operations were implemented and tested (`DeleteRange`, `InsertAt`, `Concat` in `styled/text_ops.go` / `styled/text_ops_test.go`).

Immediate next integration target:

1. style-run reporting/iteration APIs (`StyleRuns`, `EachStyleRun`).
2. paragraph synchronization (`ParagraphFromText`/`WrapAt`) on top of run primitives.

# Pre-Requisites before “Paragraphs”

1. Disentangle the API for `cords.Cord` from `CordEx`. Currently the package-structure is bad. I moved out some file into a new sub-package `cordext`, where in the future `CordEx` should live. The sub-package now breaks and is not buildable. The API in `cordext` is allowed to be broad, as it is a base-package for extension, while the base package `cords` should have a thin API, as it is geared toward end-users. The implementation in the base package should be based on the wider implementation in `cordext`. That means, dealing with `Chunk`s etc has to be moved to `cordext`. The base package should in essence use `CordEx` with `E` being `NO_EXT`.
2. Re-base sub-package `styled/` on `CordEx` instead of `cords.Cord`. This is the cleaner approach, especially when designing and implementing the “paragraph" idea. That will make the `styled.Text` API expose the extension `E` and will probably the function signatures more complicated (and possibly confusing for clients of `styled`). Let's first do the re-basing, an afterwards decide on a usable client-API.
   - Status (2026-02-26): top-level `styled` is now rebased to `cordext.CordEx[btree.NO_EXT]` for raw text storage and text ops (`DeleteRange`, `InsertAt`, `Concat`, `Section`) plus builder/paragraph raw access paths.
   - Scope caveat: `styled` sub-packages (`formatter`, `inline`, `itemized`) are not yet adapted and will not compile until they are migrated to the new raw-text type/API.
3. Tackle this TODO: “TODO: Cached subtree sizes for better performance.” (file @btree/tree.go). This is slowly becoming an issue. Make a careful step-by-step effort to introduce the substree-sizes as additional fields in the nodes. ( **Remark**: I have implemented `Weight()` for tree-nodes myself. This is not exactly the same as caching subtree sizes within inner nodes, but is more suitable for sum-trees. It should be enough to avoid deep recursion. ) With `Weight()` now available on tree nodes, the immediate pressure to add more structural caching for traversal depth has decreased.
4. Design an implement means to discovering paragraphs in text. This can (at least) be done by either:
   1. Including a “paragraph” bitfield in the nodes, similar to “lines”.
   2. Create a metric for it (like for words in sub-package `metrics`)
   3. Use the `CordEx` extension mechanism to provide a paragraph metric/summary.

## Paragraph Discovery Decision Notes (2026-02-26)

For paragraph discovery we should prefer the least invasive approach first.

Recommended first implementation:

1. Implement paragraph discovery as a dedicated analyzer/metric-style operation (not as node bitfields).
2. Build it on top of `cordext` segment iteration (`RangeTextSegment` / `EachTextSegment`) with a small separator state machine.
3. Return byte spans `[from,to)` for paragraphs, with configurable delimiter policy.

Rationale:

1. No changes to tree/node storage layout are required.
2. Fits naturally with the current `styled.Text` raw representation (`cordext.CordEx[btree.NO_EXT]`).
3. Keeps semantics explicit and testable (`\n`, `\r\n`, optional blank-line handling).

Proposed API shape (initial):

1. `type ParagraphSpan struct { From uint64; To uint64 }`
2. `type ParagraphPolicy ...` (default LF/CRLF policy first)
3. `FindParagraphs(text cordext.CordEx[btree.NO_EXT], policy ParagraphPolicy) []ParagraphSpan`

Phased strategy:

1. Phase 1: implement analyzer + tests + integration with `styled` sectioning.
2. Phase 2: add helpers like `ParagraphAt(pos)` / `ParagraphsInRange(from,to)`.
3. Phase 3 (only if needed): evaluate `CordEx` extension summaries for faster paragraph-index lookups at large scale.

Option trade-off summary:

1. Node bitfield in tree nodes: potentially fastest lookups, but highest implementation cost and strongest coupling to storage internals.
2. Metric/analyzer over segments: best first step; low risk, low complexity, good correctness surface.
3. `CordEx` extension-based paragraph summary: good medium-term optimization path if profiling shows paragraph lookup hot spots.
