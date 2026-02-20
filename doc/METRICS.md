# Metrics / Summary Extensibility for `cords`

## Question

Can clients extend a `Cord` with new summary fields/dimensions (similar to Zed SumTree usage) without rebuilding their own rope implementation from scratch?

## Short Answer

Yes, but not by mutating `Cord`'s built-in summary type in place.

`Cord` is currently fixed to:

- `btree.Tree[chunk.Chunk, chunk.Summary]`

So adding fields directly to `chunk.Summary` is not a runtime extension point for client code.

## Recommended Extension Model: Decorated Sidecar Index

Use `Cord` as canonical storage/edit structure, and attach a second summarized index built from the cord's chunk stream.

- Keep `Cord` unchanged.
- Build a sidecar tree using client-defined summary monoid + dimensions.
- Query this sidecar for seek/aggregation tasks (e.g. seek by A, accumulate B).

This gives clients a high-level extension point while reusing all rope operations.

## Why This Fits

- Clients do not have to build rope/tree/chunk logic themselves.
- Clients can define custom metrics/dimensions.
- Core `cords` storage and edit semantics remain stable.
- Matches the SumTree idea of dimension-driven seeking and summary aggregation.

## API Sketch (Conceptual)

```go
type Summarizer[S any] interface {
    Zero() S
    Add(a, b S) S
    ChunkSummary(c chunk.Chunk) S
}

type Decorated[S any] struct {
    Cord Cord
    idx  *btree.Tree[decorItem[S], S]
    s    Summarizer[S]
}

func Decorate[S any](c Cord, s Summarizer[S]) (*Decorated[S], error)
func (d *Decorated[S]) SeekBy[K any](target K, dim btree.Dimension[S, K]) (item int, acc K, err error)
func (d *Decorated[S]) PrefixSummary(itemIndex int) (S, error)
```

Notes:

- `decorItem[S]` is an internal item type that carries chunk reference + computed summary `S`.
- Dimensions remain generic and reusable as in current btree cursor design.

## Consistency / Lifecycle

Because `Cord` is persistent and immutable, the side index should also be snapshot-based.

Two workable models:

1. Wrapper edits (preferred API)
- `Decorated` exposes edit methods mirroring `Cord` operations.
- Each edit returns a new `Decorated` with updated `Cord` + updated side index.

2. Lazy rebuild
- Keep `Decorated{Cord, idx}`.
- If the cord snapshot changed relative to the stored index, rebuild index on first query.

## Tradeoffs

### Rebuild-on-edit (v1)

- simplest implementation
- always correct
- potentially higher cost on frequent edits

### Structural mirror updates (v2)

- apply split/concat/insert/cut to side index in parallel
- lower update cost
- significantly more implementation complexity

## Practical Recommendation

Start with a minimal, correct v1:

1. Build side index from `Cord.RangeChunk()`.
2. Provide seek/prefix APIs over custom summaries.
3. Rebuild index on each new decorated snapshot.
4. Add profiling and then optimize toward mirrored structural updates only if needed.

## Relation to Current Positioning Work

The new `Pos`/`CharCursor` work already demonstrates the same architectural pattern:

- base data is chunk-based rope,
- queries are powered by summary + dimension routing,
- higher-level semantics are layered without changing core rope storage.

A decorated metrics layer follows the same strategy for client-defined dimensions.

## Alternative Proposal: Core-Attached Summary Extensions

The sidecar/decorator model is practical, but if the requirement is that extension
data lives directly in every leaf/internal summary, we should move the extension
point into the core cord summary model.

### Key Idea

Refactor core items so each leaf item carries a schema-aware summary, and internal
node summaries aggregate those values as usual.

Conceptually:

```go
type cordItem struct {
    chunk   chunk.Chunk
    summary Summary // core + extension slots
}
```

Where `Summary` contains:

- core fields (`Bytes`, `Chars`, `Lines`)
- extension slot storage (for example fixed numeric slots)

A `SummarySchema` defines:

- how extension slots are computed for each chunk/item,
- how slots combine (monoid rules),
- which seek dimensions are exposed over those slots.

### Why This Satisfies the Requirement

- extension data is physically stored inside leaf summaries and internal summaries,
- all queries run on the canonical cord tree (no separate side index),
- clients reuse `cords` APIs instead of building a separate btree/chunk stack.

### Constraints / Design Rules

1. Schema is fixed per cord/tree instance.
2. Cross-cord operations (`Concat`, `Insert`, etc.) require schema compatibility.
3. Extension slot values should be monoidal and cheap to aggregate.
4. Initial version should prefer fixed-size numeric slots for predictable performance.
5. Schema identity/fingerprint must be tracked and checked on tree operations.

### Implementation Direction (Phased)

1. Add `SummarySchema` abstraction and schema fingerprinting.
2. Introduce internal `cordItem` and move `Cord` tree type to `Tree[cordItem, Summary]`.
3. Keep default schema equivalent to current behavior (`Bytes/Chars/Lines` only).
4. Add extension registration and extension-driven dimensions.
5. Enforce schema compatibility checks for cross-cord structural operations.

### Tradeoff vs Sidecar Model

- Pros:
  - first-class extensions inside canonical tree summaries,
  - no synchronization concerns between main tree and side index.
- Cons:
  - deeper core refactor,
  - schema compatibility and migration complexity,
  - larger impact surface on internals and possibly on package APIs.

# Further Considerations

I am unsure whether this extension point should live in the cord implementation or pushed down into the btree implementation. Extension summarizers should be allowed read-only access to items I. During re-structuring operations (i.e. when summaries are to be calculated afresh) the extension should be called to recalculate (with item I as one of the arguments). Instead of making the new summarizer-extension E dependent on types I and S, it would be more suitable to have an interface `Extendable`, which would signal that an extension may attach to this item/summary combination. Or the other way round: an item-type I may have an extension-point `Extend(Extension)` for an interface type `Extension`. A realistic example would be a client wanting to use cords as a data-structure for an editor. This client would need to have bookkeeping for a row/column-cursor, i.e. a `Point` like in Zed. The extension mechanism should be able to make this possible and to let the extension re-calculate the `Point` whenever the cord structure changes. Or the client is able to calculate a `Point` from a `Pos`.


## Placement Decision: `cords` vs `btree`

### Recommendation

Split responsibilities:

1. `btree` remains generic and minimal:
   - node structure and persistence,
   - summary monoid aggregation,
   - dimension-based seek/cursor mechanics.

2. `cords` hosts the extension mechanism:
   - extension registration,
   - schema composition (core + extension fields),
   - compatibility checks across cord operations.

### Rationale

- `btree` already provides the required generic abstraction (`S` + monoid + dimensions).
- Runtime extension registration is domain-specific and text-oriented; this belongs in `cords`. (**Remark**: I am not sure this is true. A cord is more or less a `btree` with `Chunk` as item-type and some extra calculations for rune boundaries. The extension mechanism may be usefule for any sum-tree in the context of text/editing/typesetting. Zed uses its sum-trees for all kinds of tasks. We ourselves will use another sum-tree for text styling. A client needs to know the item-type `I` in order to attach a fitting extension type.)
- Keeping extension policy out of `btree` avoids coupling generic tree code to text-editor semantics. (**Remark**: `Point` is just an example. Let's use another one: We will use a sum-tree to project styles (i.e., “bold”, “italic”) onto the text of a cord. However, clients may want to extend the styling mechanism and introduce other styling options.)

## Extension API Shape (Conceptual)

### Why not `I.Extend(...)` or marker-only `Extendable`

A marker interface alone does not define recomputation behavior.  
An `I.Extend(...)` mutation-style hook would push mutable policy into items and
work against persistence/immutability assumptions. (**Remark**: `I.Extend()` was meant in the sense of `I.AttachExtension(…)` to allow the `FromItem(I)` call from below. However, I prefer the cord or btree type to host the extension schema, but I am unsure of how to align the types without the signatures becoming too complicated.)

### Prefer schema-driven extension contracts

Extensions should have read-only access to items and deterministic combine rules:

```go
type Extension[I any, ES any] interface {
    // Leaf/item projection (read-only access to item)
    FromItem(item I) ES
    // Internal aggregation (monoid combine)
    Add(left, right ES) ES
}
```

`cords` composes:

- core summary fields (bytes/chars/lines),
- one or more extension summaries.

During split/merge/rebalance/path-copy, summary recomputation naturally invokes
`FromItem` at leaf rebuild and `Add` at internal rebuild.

## `Point` Example (Editor Use Case)

A client can define an extension summary for point bookkeeping (row/column-like
state composition), then expose:

- `Pos -> Point` via prefix summary accumulation,
- optionally `Point -> Pos` via a dedicated dimension.

This mirrors the SumTree style: seek by one dimension, accumulate another.

## Compatibility and Safety

- Each cord snapshot has one fixed schema. (**Remark**: I am unsure where this requirement comes from. Extensions have to have suitable types for `I` and `S`. What is the purpose of another schema declaration?)
- Cross-cord operations (`Concat`, `Insert`, etc.) require schema fingerprint match.
- Extension code receives read-only items only. (**Remark**: Conceptually that is true. However, always making a copy for each call would be too expensive. We could make the 'ro' rule a convention/recommendation in a first draft. Let's decide later.)
- Recalculation happens through normal summary rebuild paths; no side mutations. (**Remark**: yes)


# Reconsideration
 `btree`-First Extension Primitive (Updated Findings)

After reviewing the remarks in “Further Considerations”, the previous placement decision
should be refined.

### Updated Conclusion

A better long-term split is:

1. `btree` provides the generic extension primitive.
2. `cords` provides text-specific defaults and convenience APIs.

This keeps extension capability reusable for non-cord sum-tree use-cases (for example
style projection trees and other editor/typesetting indexes), while preserving an ergonomic
API for cord clients.

### Core Mechanism (Conceptual)

Instead of relying only on `item.Summary()`, tree config can carry an item summarizer/projection:

```go
type ItemSummarizer[I any, S any] interface {
    SummaryOf(item I) S
}
```

Tree maintenance then uses:

- `SummaryOf(item)` when leaf summaries are rebuilt,
- monoid `Add(left,right)` for internal aggregation.

This directly satisfies the requirement that extension logic is called during split/merge/
rebalance/path-copy recalculation and receives read-only access to item `I`.

### On `Extendable` / `I.AttachExtension(...)`

- Marker-only `Extendable` is too weak to define recomputation behavior.
- `I.AttachExtension(...)` is possible, but tends to push policy into items and complicates
  immutability expectations.
- Hosting extension policy at tree/schema level is cleaner and keeps items simple.

### On Schema Requirement

The schema/fingerprint requirement is conditional:

- If extension set is fixed by static Go type `S`, compile-time typing already enforces much
  of compatibility.
- If extensions are runtime-configurable with the same `S` shape, a runtime schema ID/fingerprint
  remains useful for operation compatibility checks (`Concat`, `Insert`, etc.).

So “schema” is not mandatory in all variants; it is primarily needed for runtime-pluggable
configurations.

### Read-Only Access and Cost

Read-only access should be a contract/convention, not implemented via copying item values on each
callback. Per-call copying would be too expensive for hot rebuild paths.

### Revised Option Set

1. `cords`-only extension host
- smallest immediate refactor, less reusable.

2. `btree`-native summarizer extension primitive
- broader reuse, deeper btree refactor.

3. Hybrid (preferred)
- implement summarizer primitive in `btree`;
- expose text/rune/position-focused convenience in `cords`.

This hybrid path preserves prior design goals while addressing the extensibility concerns raised
in the remarks.

## Implementation Plan (Hybrid Model)

The implementation follows the revised hybrid approach:

- generic extension primitive in `btree`,
- text-oriented convenience and schema ergonomics in `cords`.

### 1. Design Freeze (Mini RFC)

1. Decide extension payload shape for v1.
2. Recommended v1: fixed numeric slots (`uint64`) plus core summary fields.
3. Confirm compatibility rule: cords with different schema IDs cannot be structurally combined.

### 2. `btree` Primitive Refactor

1. Relax tree item constraint from `I SummarizedItem[S]` to `I any`.
2. Add config hook for item projection:
   - `ItemSummary func(I) S`
   - keep monoid combine unchanged.
3. Constructor compatibility:
   - if `ItemSummary == nil`, adapt from `item.Summary()` when available.
4. Replace internal `item.Summary()` calls with `t.itemSummary(item)`.
5. Keep existing `btree` tests green.

### 3. `cords` Summary Schema Core

1. Introduce internal schema-aware summary type in `cords`:
   - core (`Bytes`, `Chars`, `Lines`),
   - extension payload/slots.
2. Add schema object:
   - schema ID/fingerprint,
   - item projectors,
   - combine logic.
3. Make `Cord` carry schema metadata.
4. Default schema reproduces current behavior exactly.

### 4. Migrate `cords` Tree Wiring

1. Switch `Cord` tree type from `Tree[chunk.Chunk, chunk.Summary]` to schema-backed summary.
2. Keep existing dimensions (`Byte`, `Char`, `Line`) mapped to core summary fields.
3. Verify byte/rune APIs (`Pos`, cursor, split/report wrappers) remain behavior-compatible under default schema.

### 5. Public Extension API in `cords`

1. Add schema builder / extension registration API.
2. Extension contract (read-only item access):
   - item projection (`FromItem`-like),
   - monoid combine (`Add`-like).
3. Add constructors using schema:
   - `FromStringWithSchema(...)`
   - `NewBuilderWithSchema(...)`.

### 6. Compatibility and Safety

1. Enforce schema compatibility checks in structural ops (`Concat`, `Insert`, ...).
2. Add `ErrIncompatibleSchema`.
3. Keep read-only item access as contract/convention (no per-call copies in hot paths).

### 7. First End-to-End Extension Example

1. Implement a `Point`-style extension (`Pos -> Point`).
2. Optionally add inverse mapping (`Point -> Pos`) via a dedicated dimension.
3. Document example usage and guarantees.

### 8. Performance and Hardening

1. Benchmarks with and without extensions.
2. Allocation checks on summary recomputation paths.
3. Fuzz/property tests for schema compatibility and extension invariants.
