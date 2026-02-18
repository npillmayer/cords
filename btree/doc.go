/*
Package btree provides an experimental, rope-focused B+ sum-tree backend for
cords.

The package is intentionally not a generic map/set container. It is specialized
for sequence storage with positional editing and persistent (copy-on-write)
updates. The implementation is built in parallel to the current binary rope so
we can validate correctness and performance before a backend switch.

Current status:
  - package skeleton and invariants framework,
  - summary and dimension interfaces,
  - item-to-summary linkage at the type level (`item.Summary()`),
  - distinct `leafNode` and `innerNode` representations,
  - backend-selectable node storage (`dynamic` default, `btree_fixed` build tag),
  - tree API surface and summary-guided cursor seek,
  - recursive path-copy insert with split propagation,
  - path-copy split with subtree sharing (fallback to rebuild for strict edge cases),
  - baseline concat implementation (correctness-first, not optimized),
  - operation stubs for cut style workflows.

# BSD License

Copyright (c) Norbert Pillmayer <norbert@pillmayer.com>

Please refer to the License file for details.
*/
package btree

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
