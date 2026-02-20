package btree

// SumExtension defines extension summary behavior maintained alongside core
// tree summaries.
//
// Implementations are expected to be pure and deterministic:
//   - `FromItem` should treat inputs as read-only and produce the same output
//     for equal inputs.
//   - `Add` and `Zero` should form a monoid for E:
//     Add(Add(a,b),c) == Add(a,Add(b,c))
//     Add(Zero(),x) == x == Add(x,Zero())
//
// `MagicID` identifies extension semantics for compatibility checks in
// cross-tree structural operations (for example Concat). It should be stable
// for a given extension configuration and should change when semantics change.
type SumExtension[I SummarizedItem[S], S, E any] interface {
	// MagicID returns a stable identifier for extension semantics.
	MagicID() string
	// Zero returns the neutral element for extension aggregation.
	Zero() E
	// FromItem projects one leaf item and its base summary into extension space.
	FromItem(I, S) E
	// Add combines two extension summaries.
	Add(E, E) E
}

// NO_EXT is a marker type used when no extension summary is configured.
type NO_EXT struct{}
