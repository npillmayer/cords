# Rope-Specific B-Tree Proposal

## Goal

Replace or complement the current binary-tree rope core with a rope-specific B+ tree that:

- preserves immutable/persistent semantics (copy-on-write),
- supports efficient positional text edits at scale,
- allows explicit control over internal node metadata,
- maps cleanly to the existing `cords` API (`Concat`, `Split`, `Insert`, `Cut`, `Substr`, `Reader`, summary/dimension queries).

## Why Build Our Own

Generic Go B-tree libraries are excellent containers, but they hide internal nodes and split/merge behavior.  
For a rope, we need direct control over:

- per-child byte-weight prefix sums,
- optional extension aggregates per subtree,
- chunking policy and leaf storage,
- path-copy update mechanics for persistence.

That requires a purpose-built tree structure.

## Expected Code Size

For production-quality code (not including optional advanced features):

- Core B+ tree mechanics: `~400-700` LOC
- Rope operations/indexing glue: `~250-500` LOC
- Persistence/path-copy support: `~150-300` LOC
- Total core: `~800-1500` LOC
- Tests/fuzz/property checks: often `1000+` LOC

This is still much smaller than generic container libraries because we can omit map/set features, comparator logic, rich iterator surfaces, and deletion freelists.

## Minimal Data Model

Current scaffold (`btree/`) uses:

- `Tree[I, S]`
  - `cfg Config[S]`
  - `root treeNode[I, S]`
  - `height int`
- `Config[S]`
  - `Monoid SummaryMonoid[S]`
- `SummarizedItem[S]`
  - every item must provide `Summary() S` (compile-time item/summary binding)
- `treeNode[I, S]` interface
  - `isLeaf() bool`
  - `Summary() S`
- `leafNode[I, S]`
  - `summary S`
  - `items []I`
- `innerNode[I, S]`
  - `summary S`
  - `children []treeNode[I, S]`

Fixed structural constants:

- `max children/items per node = 12`
- `target minimum occupancy = 6` (used by balancing helpers)

## Complexity Targets

Assuming bounded fill and balanced tree:

- `Index`: `O(log_B n)`
- `Split`: `O(log_B n)`
- `Concat`: amortized `O(log_B n)` (or better for adjacent balanced roots)
- `Insert`/`Cut`: `O(log_B n + k)` where `k` is affected local chunks
- Sequential read/report: `O(m)` over bytes emitted plus leaf navigation overhead

Compared with binary ropes, the main win is lower height (`log_B n` vs `log_2 n`) and far fewer pointer hops.

## Required Core Operations

### 1. Positional descent

Descend through children with index/dimension routing (to be implemented in core edit/seek algorithms).

### 2. Leaf split/chunk split

Split within a chunk by byte index, then rebalance if node overflows.

### 3. Internal split and upward propagation

When a child overflows, split node around median and push one sibling upward.

### 4. Merge/borrow on delete-like ops

`Cut` and some split-combine paths need underflow handling:

- borrow from sibling when possible,
- otherwise merge siblings and adjust parent.

### 5. Path-copy persistence

On write operations, clone nodes on the search path only; untouched subtrees remain shared.

## API Mapping Strategy

Keep public API mostly stable:

- `FromString`: build one or few leaves.
- `Concat`: root-level join with rebalance.
- `Split(i)`: descend, split leaf, clone path, return left/right roots.
- `Insert(c, i)`: split + concat three-way.
- `Cut(i, l)`: split twice + concat remaining parts.
- `Substr(i, l)`: split/select.
- `Reader`: cursor over leaves/chunks without full materialization.

`Leaf` interface can remain, though a chunk type optimized for byte slices may improve performance.

## Summary/Extension Integration Plan

Phase 1:

- Use built-in summary fields (`Bytes`, `Chars`, `Lines`) for core counts.
- Expose prefix/seek operations through dimensions and cursors.

Phase 2:

- Add optional extension aggregates where clients need domain-specific summaries.
- Update extension aggregates incrementally during split/merge/path-copy.
- Speed up extension-based range queries through cached subtree extension summaries.

## Phased Implementation Plan

1. Build v1 tree skeleton
- Node structs, invariants, positional descent, split/merge primitives.

2. Wire minimal cord ops
- `Len`, `Index`, `Report`, `Reader`, `Concat`, `Split`.

3. Add edit ops
- `Insert`, `Cut`, `Substr`.

4. Persistence hardening
- Strict path-copy semantics + sharing tests.

5. Summary/extension compatibility
- Keep summary and extension aggregation correct across all edit operations.

6. Optimize and tune
- Chunk-size tuning, borrow/merge heuristics, benchmark-guided cleanup.

## Testing Strategy

Must-have tests:

- Invariants after every edit (node bounds, fill factors, byte sums, height consistency).
- Behavioral parity against current implementation for core API.
- Persistence tests: original cords remain unchanged after edits.
- Randomized operation sequences with reference model (`string`/`[]byte`) checks.
- Fuzzing for split/concat/cut boundaries.

## Risks

- Underflow/merge edge cases can be subtle and bug-prone.
- Byte indexing vs UTF-8 semantic boundaries still requires clear documentation.
- Over-eager optimization may complicate correctness early.

Mitigation: start with correctness-first v1, instrument invariants heavily, and benchmark before adding advanced caches.

## Recommended Starting Point

Implement a rope-specific B+ tree directly (not a generic container abstraction), with:

- moderate fanout (currently 12, aligned with Zed's `TREE_BASE=6` shape),
- fixed chunk targets (e.g. 2 KiB),
- strict invariant checks in debug/test builds,
- minimal API-compatible surface first.

This yields full node control while keeping scope manageable.

## V2 Proposal (SumTree-Aligned)

Based on the Zed SumTree model, the proposal should be tightened in five ways.

### 1. Make summaries first-class from day 1

Do not treat aggregate metadata as an optional optimization.  
Each node should carry a composable summary (monoid) that is always maintained.

- Internal nodes aggregate child summaries.
- Leaves aggregate item/chunk summaries.
- Core operations (`split`, `merge`, `concat`, path-copy updates) must update summaries as part of correctness.

### 2. Add a dimension-based cursor API

Introduce explicit seek dimensions and a reusable cursor abstraction.

- Example dimensions: bytes, runes, lines, UTF-16 units, points.
- Cursor seeks by one dimension while accumulating others during descent.
- This enables one-pass conversions (for example offset-to-line/column style queries).

### 3. Store multiple items per leaf node

Prefer packed leaves with multiple text chunks/items and per-item summaries.

- Better cache locality and lower pointer overhead.
- Fewer node allocations on edits.
- Natural fit for chunk splitting/coalescing policies.

### 4. Separate generic tree core from rope surface

Refactor architecture into:

- `sumtree` core: summarized persistent B+ tree, dimensions, cursor.
- `rope` adapter: text chunks, rope API compatibility, text-specific helpers.

This keeps rope use-cases fast now while leaving room for additional summarized indexes later.

### 5. Adjust implementation estimate if V2 is immediate

If V2 features are included from the start:

- Core summarized B+ tree + persistence + cursor dimensions: `~1400-2400` LOC
- Rope adapter and compatibility layer: `~300-700` LOC
- Tests/fuzz/property checks: `~1500+` LOC

The original `~800-1500` LOC estimate remains reasonable for a rope-only v1 without generic dimensions.

## Revised Rollout (Recommended)

1. Build summarized B+ tree core (mandatory summary monoid + invariants)
2. Add path-copy persistence and structural edit primitives
3. Implement dimensioned cursor/seeking
4. Implement rope adapter mapped to existing `cords` API
5. Add compatibility tests against current behavior
6. Optimize chunk sizing and dimension set by benchmark data

## Practical Default

For the first delivery, keep the dimension set small (bytes + lines), but keep the
API generic enough to add more dimensions later without structural rewrites.

## Implementation Snapshot (Current `btree/`)

Implemented:

- Generic monoid summary contract (`SummaryMonoid`).
- Type-level item-summary linkage (`SummarizedItem[S]`).
- Distinct leaf/internal node types.
- Tree invariants checker (`Check`).
- Text defaults: `TextChunk`, `TextSummary`, `TextMonoid`, byte/line dimensions.
- Summary-guided cursor seek (`NewCursor`, `Seek`) for generic dimensions.
- Internal mutation helper layer (clone/path-copy helpers, summary recomputation,
  slice child/item mutation helpers, occupancy checks).
- Leaf-local mutation primitives (`insertIntoLeafLocal`, `splitLeaf`) with
  promoted-sibling output for upcoming parent-level insert propagation.
- Recursive path-copy `InsertAt` with upward split propagation (including root splits).
- `SplitAt` now performs path-copy structural splitting with subtree sharing.
  - No rebuild fallback path remains.
- `Concat` now uses a structural, height-aware join.
  - It path-copies only affected boundary paths and shares untouched subtrees.
- Fixed-array node backend is the active implementation.
- Mutation primitives perform in-place shifts on fixed storage
  (no per-node slice reallocation on local edits).
- Backend-specific invariants validate fixed-storage view/occupancy consistency
  during `Check()`.

Not implemented yet:

- Full structural edit algorithms at tree level (recursive split/merge/borrow/rebalance/path-copy).
- Real cursor seek traversal.
- Byte-offset/range indexing primitives.
- Metrics/styled-text integration over real edits.

Current performance notes:

- `Len()` currently computes item count by tree traversal (`O(n)` in item count).
- Height is cached on `Tree`; item count is intentionally not cached at this stage.

## Zed Reference Sources

For this proposal, the following Zed rope/sum-tree sources are available and suitable
for direct cross-checking:

- Zed repository rope crate: `crates/rope`  
  <https://github.com/zed-industries/zed/tree/main/crates/rope>
- Zed repository sum-tree crate: `crates/sum_tree`  
  <https://github.com/zed-industries/zed/tree/main/crates/sum_tree>
- Published API/source docs for rope:  
  <https://docs.rs/zed-rope/latest/zed_rope/>
- Published API/source docs for sum-tree:  
  <https://docs.rs/zed-sum-tree/latest/zed_sum_tree/>

## Comparison: Current Draft vs Zed Model

This section compares the **current** `btree/` implementation with Zed's model.

### Areas now aligned

- Monoid-based summary composition.
- Item/summary linkage at the type level (`item.Summary()`).
- Distinct node variants (`leafNode` vs `innerNode`).
- Fixed-capacity node shape aligned to a `TREE_BASE=6` style layout.

### Remaining deltas

- No `Context`-parameterized summaries yet (intentionally deferred).
- No rich seek-target/bias/path-stack cursor model yet.
- No delete/cut/merge/borrow rebalancing yet.
- No cached item count (current `Len()` is traversal-based).

## Recommended Next Steps

1. Implement delete/merge/borrow primitives and re-tighten occupancy policies.
2. Define path-copy invariants explicitly before delete/rebalance logic lands.
3. Upgrade cursor API toward richer target+bias semantics.
4. Add efficient positional dimensions needed by rope API (bytes first).
5. Re-evaluate adding `Context` when extension/styled-text operations start.

Scope note:

- Public code/docs are available for inspection.
- Private/internal repositories are not accessible.

## Implementation Plan (Core Tree Operations)

The current scaffold is ready to begin implementing core tree operations.

1. Lock invariants and helper primitives
- Add internal helpers for node cloning (path-copy), summary recomputation,
  child/item slice insert/remove, and occupancy checks.
- Assert invariants around each helper.

2. Implement leaf-level mutation primitives
- Insert item(s) into a leaf at local index.
- Split overflowing leaves into left/right siblings.
- Return promoted sibling information to parent-level logic.

3. Implement recursive insert with upward split propagation
- Internal insert returns either updated child only, or updated child + promoted sibling.
- Handle root split by creating a new `innerNode` root.
- Wire this into public `InsertAt`.

4. Implement `SplitAt` on item index
- Descend to split point with path-copy.
- Produce two valid trees.
- Normalize roots/heights and recompute summaries.

5. Implement `Concat` as tree join
- Start with a simple and correct baseline:
  - empty-side short-circuit clone,
  - join by height alignment,
  - rebalance on affected path.
- Optimize join strategy later if needed.

6. Build correctness harness before optimization
- Extend tests with randomized insert/split/concat sequences.
- Run invariants after each operation.
- Add reference-model checks against plain slice/string behavior.
- Add persistence checks (original trees unchanged).

7. Implement cursor seek after structural ops stabilize
- Implement `Seek` over summaries in cursor.
- Start with minimal semantics (first item reaching/exceeding target).
- Add boundary bias and richer seek targets later.

8. Defer rope byte-index mapping to adapter layer
- Keep tree operations item-indexed initially.
- Map byte offsets via summary dimensions once cursor traversal is implemented.

## Proposal: Fixed-Array Nodes (ArrayVec Style)

This proposal replaces per-node dynamic slices with fixed-capacity arrays plus
explicit logical lengths, similar to Rust `ArrayVec` usage.

### Motivation

- Avoid per-node backing-array allocations for `children` and `items`.
- Keep node payload contiguous in memory (better locality, fewer pointer hops).
- Make occupancy invariants explicit and cheap (`0 <= n <= cap`).
- Keep split/borrow/merge operations bounded to small memmoves.

### Core Design

Use compile-time capacities for node storage:

- `maxChildren = 12` (aligned with Zed `TREE_BASE=6` shape)
- `minChildren = 6`
- `maxLeafItems = 12` (initially same as internal degree for simplicity)
- `minLeafItems = 6`

Node layouts:

```go
type leafNode[I SummarizedItem[S], S any] struct {
	summary S
	n       uint8
	itemStore [maxLeafItems+1]I
	items     []I
}

type innerNode[I SummarizedItem[S], S any] struct {
	summary    S
	n          uint8
	childStore [maxChildren+1]treeNode[I, S]
	children   []treeNode[I, S]
}
```

Notes:

- `n` is logical occupancy; `items`/`children` are views into fixed storage.
- The extra `+1` slot supports transient overflow before split without immediate reallocation.
- No per-child summary cache is kept in the current implementation; parents
  aggregate directly from `child.Summary()` after local edits.

### API/Configuration Impact

Go has no const generics, so fixed capacities are effectively compile-time.
That means:

- Runtime `Degree`/`MinFill` knobs are removed from `Config`.
- Tunability moves to internal constants instead of per-tree runtime knobs.

Recommended approach:

- Keep capacities fixed and internal for now.
- Re-evaluate exposing tuning knobs only after benchmarks justify it.

### Operation Changes

Insertion/removal inside a node uses in-place shifts over array windows:

```go
// Insert k values at idx
copy(items[idx+k:n+k], items[idx:n])
copy(items[idx:idx+k], values)
n += k
```

```go
// Remove [from,to)
copy(items[from:n-(to-from)], items[to:n])
n -= (to - from)
```

Split policy:

- On overflow (`n == cap+1` transiently), split around midpoint.
- Move upper half into a newly allocated sibling node.
- Recompute summaries for left/right; propagate sibling upward.

Path-copy cloning:

- Clone by value copy (`clone := *orig`) for each node on the edited path.
- This copies full fixed arrays each time; cost is predictable and bounded.

### Tradeoffs

Pros:

- Fewer allocations and less GC pressure per mutation.
- Better cache locality.
- Simpler occupancy checks and no slice-capacity corner cases.

Cons:

- Cloning copies full arrays, even when sparsely filled.
- Larger `I` values inflate node copy cost.
- Degree is no longer runtime-tunable.

Mitigations:

- Keep `I` small (descriptor/pointer/value-header), not large inline payloads.
- Use chunk objects for heavy text storage.
- Start with conservative capacities (`12`) and tune by benchmark data.

### Migration Plan

1. Keep fixed-array backend as the only active backend.
2. Add delete/merge/borrow operations and re-tighten occupancy policies.
3. Benchmark edit/read workloads and allocation profiles.

## Delete Rebalancing Plan (Borrow/Merge)

1. Lock invariants
- Enforce occupancy in `Check()`:
  - non-root leaf: `fixedBase <= len(items) <= fixedMaxLeafItems`
  - non-root inner: `fixedBase <= len(children) <= fixedMaxChildren`
- Enforce root normalization constraints:
  - empty tree: `root == nil && height == 0`
  - root leaf must be non-empty
  - root inner with one child is invalid (must be collapsed)
- Keep internal inconsistencies as assertions; keep returned errors for input misuse.

2. Add single-item delete primitive
- Add `DeleteAt(index int) (*Tree, error)` as first delete API.
- Validate index as input error (`0 <= index < Len()`).
- Implement compositionally via `SplitAt(index)`, drop one item via `SplitAt(1)`, and `Concat`.
- Preserve persistence/path-copy semantics without introducing merge/borrow logic yet.

3. Introduce recursive delete core
- Add internal recursive delete with path-copy descent.
- Return `underflow` signal to parent when non-root node drops below min occupancy.
- Recompute summaries on the modified path.

4. Implement borrow (redistribution) helpers
- Leaf borrow:
  - borrow from left sibling: move last item left->right.
  - borrow from right sibling: move first item right->left.
- Inner borrow:
  - transfer one child pointer from sibling to underfull node.
- Recompute summaries on both siblings and parent.

5. Implement merge helpers
- Leaf merge: concatenate neighboring leaves and remove one child from parent.
- Inner merge: concatenate child arrays and remove one child from parent.
- Bubble parent underflow upward when needed.

6. Centralize parent-side rebalancing policy
- On child underflow:
  - try borrow from left sibling if it has spare occupancy,
  - else borrow from right sibling,
  - else merge with a sibling (stable side preference).
- Keep this in one helper to avoid divergent edge-case logic.

7. Root normalization after delete
- If root becomes empty: `root=nil`, `height=0`.
- If root is inner with one child: promote that child and decrement height.
- Keep root normalization explicit at operation end.

8. Tests before range delete
- Borrow left/right for leaves and inners.
- Merge leaf and inner paths.
- Cascading underflow to root shrink.
- Persistence guarantees (original tree unchanged).
- Input validation (`index<0`, `index>=Len()`).

9. Add range delete
- Add `DeleteRange` using either:
  - split/delete/concat composition, or
  - batched recursive delete.
- Choose based on benchmark and complexity tradeoff.
