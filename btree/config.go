package btree

import "fmt"

const (
	// DefaultDegree is the fixed max fanout used by the current tree implementation.
	DefaultDegree = 12
	// DefaultMinFill is the fixed lower occupancy bound used by balancing helpers.
	DefaultMinFill = 6
)

// SummarizedItem ties a leaf item to its summary type at compile time.
type SummarizedItem[S any] interface {
	Summary() S
}

// SummaryMonoid defines how summaries are aggregated up the tree.
//
// For summaries s, t, u, Add should be associative:
//
//	Add(Add(s, t), u) == Add(s, Add(t, u))
//
// and Zero should be the neutral element:
//
//	Add(Zero(), s) == s == Add(s, Zero())
type SummaryMonoid[S any] interface {
	Zero() S
	Add(left, right S) S
}

// Config configures a rope-focused B+ sum-tree.
type Config[S any] struct {
	// Monoid aggregates summaries up the tree.
	Monoid SummaryMonoid[S]
}

func (cfg Config[S]) normalized() Config[S] {
	return cfg
}

func (cfg Config[S]) validate() error {
	cfg = cfg.normalized()
	if cfg.Monoid == nil {
		return fmt.Errorf("%w: monoid is required", ErrInvalidConfig)
	}
	return nil
}
