# Cords Base Package Overview

## Scope

This review covers the root `cords` package only (top-level files in the module root), excluding sub-packages such as `metrics/`, `styled/`, and `textfile/`.

## Purpose Of The Package

`cords` is an immutable rope/cord implementation for text editing and text-processing workflows where repeated concatenation/splitting/substring operations must remain efficient for larger texts.

The package positions itself as:

- A persistent (copy-on-write) text structure.
- A better fit than contiguous strings for editing-heavy use cases.
- A host for composable text metrics that can be computed over fragments.

## Core Data Model

The structure is a binary tree with these key invariants:

- Internal node weight equals total byte length of its left subtree.
- Leaf node weight equals fragment byte length.
- Root is wrapped as an internal node and usually stores the full cord length in `root.weight`.
- Leaves implement a pluggable `Leaf` interface (`Weight`, `String`, `Substring`, `Split`).

Default leaf type is `StringLeaf`, but leaf storage/behavior is intentionally extensible.

## Implementation Pattern

The implementation combines:

- Persistent data-structure semantics via selective cloning (`clone`, `cloneNode`) on edits.
- Structural editing primitives (`Concat`, `Split`, `Cut`, `Insert`, `Substr`) layered on tree operations.
- Tree balancing by rotation (`rotateLeft`, `rotateRight`) with AVL-like thresholding (`balanceThres = 1`).
- Traversal-first algorithms (`traverse`, `index`, `substr`) reused across user-facing operations.

This is a hand-rolled tree implementation with explicit invariant maintenance, not a reused B-tree/LLRB dependency.

## Public API Shape (Root Package)

Main entry points are:

- Construction: `FromString`, `NewBuilder`, `Builder.Append`, `Builder.Prepend`, `Builder.Cord`.
- Editing: `Concat`, `Insert`, `Split`, `Cut`, `Substr`.
- Access: `Len`, `IsVoid`, `Index`, `Report`, `String`, `Reader`, `FragmentCount`, `EachLeaf`, `RangeLeaf`.
- Metrics: `ApplyMetric`, `ApplyMaterializedMetric` via `Metric`, `MaterializedMetric`, `MetricValue`.
- Debug: `Cord2Dot`, `Dotty`.

The API is intentionally byte-oriented, not rune/grapheme-oriented.

## Extension Points

Primary extension hooks:

- `Leaf` interface for custom fragment representations (not only plain strings).
- Metric framework:
  - `Metric` for fold-like analysis over text fragments.
  - `MaterializedMetric` for analysis that also emits a span cord.
  - `MetricValue` contract with unprocessed boundary bytes support.

Design implication: clients can adapt both storage format (leaf level) and analysis behavior (metric level) without changing core rope algorithms.

## Efficiency Considerations

Expected behavior (assuming balanced trees):

- Concatenation/split/edit operations are intended near `O(log n)` in tree height.
- Sequential traversal/reporting is `O(k)` for bytes visited plus traversal overhead.
- `String()` is expensive (`O(n)` materialization + allocation).

Notable implementation details affecting cost:

- Balancing occurs after concatenation and split, improving asymptotic behavior under repeated edits.
- The root-left wrapper invariant simplifies length accounting but adds structural indirection.
- `Reader.Read` repeatedly calls substring traversal; good for streaming, but each read still walks the tree for that slice.
- Everything is byte-counted; Unicode-safe semantic operations (runes/graphemes) are delegated to higher layers or custom metrics.

## Correctness And Robustness Notes

The package has strong internal consistency checks, but many are `panic`-based and can surface hard failures if invariants are violated.

Potential issues in current root package:

1. `Builder.Cord` uses a value receiver (`func (b Builder) Cord() Cord`), so `b.done = true` does not persist to the original builder instance.
   - Documented behavior says adding after `Cord()` is illegal, but the guard likely does not work as intended.
2. `Insert` begins with `dump(&cord.root.cordNode)` before nil checks.
   - Inserting into a void cord at index `0` should be valid by API intent, but this debug call can panic when `cord.root == nil`.
3. Several operations rely on internal panics for impossible states.
   - Reasonable during development, but library consumers may prefer fully error-returning behavior.

## Documentation State (Root)

Current docs are strong on motivation and conceptual background (`README.md`, `doc.go`) but light on practical API contracts:

- Good: rationale, rope context, high-level complexity discussion.
- Missing: explicit guarantees on persistence semantics, balancing behavior, Unicode caveats, and panic/error boundaries.
- `doc/OVERVIEW.md` and `doc/DATATYPE.md` were empty prior to this review.

## Overall Assessment

The base package is a thoughtful rope implementation oriented to immutable text editing and metric-friendly processing. The architecture is coherent: persistent tree edits, rebalancing, and extensible leaf/metric contracts all align with the stated goals.

Primary risks are concentrated in API edge behavior (panic paths and a few receiver/debug-call pitfalls), not in conceptual design. Addressing those would improve reliability without changing the package’s core model.
