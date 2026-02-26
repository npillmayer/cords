/*
Package btree provides an experimental, rope-focused B+ sum-tree backend for
cords.

The package is intentionally not a generic map/set container. It is specialized
for sequence storage with positional editing and persistent (copy-on-write)
updates. The implementation is built in parallel to the current binary rope so
we can validate correctness and performance before a backend switch.

Current status:
  - tree and invariants framework,
  - summary and dimension interfaces,
  - item-to-summary linkage at the type level (`item.Summary()`),
  - optional extension summaries (`E`) via `SumExtension[I,S,E]`,
  - extension-aware node aggregation and recomputation,
  - distinct `leafNode` and `innerNode` representations,
  - fixed-array node storage with dynamic views (`items`/`children`) over inline buffers,
  - tree API surface and summary-guided (`Cursor`) / extension-guided (`ExtCursor`) seek,
  - in-order iteration (`ForEachItem`) and ranged iteration (`ItemRange`),
  - prefix aggregation for summaries (`PrefixSummary`) and extensions (`PrefixExt`),
  - recursive path-copy insert and delete with sibling rebalance,
  - path-copy split with subtree sharing,
  - structural, height-aware concat/join with path-copy updates,
  - public editing operations: `InsertAt`, `DeleteAt`, `DeleteRange`, `SplitAt`, `Concat`,
  - extension compatibility checks for cross-tree concat (`MagicID`),
  - ongoing hardening and cleanup while preparing backend integration.

Extension model:
  - `S` is the base tree summary used by core monoid aggregation and dimensions.
  - `E` is optional extension summary state, maintained in parallel to `S`.
  - Clients provide extension behavior through `SumExtension[I,S,E]`.
  - `Concat` requires extension compatibility: both trees must expose the same
    extension `MagicID`.

# BSD License

Copyright (c) Norbert Pillmayer <norbert@pillmayer.com>

Please refer to the License file for details.
*/
package btree

import "github.com/npillmayer/schuko/tracing"

// tracer traces with key 'cords.btree'.
func tracer() tracing.Trace {
	return tracing.Select("cords.btree")
}

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
