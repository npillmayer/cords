package btree

import "fmt"

const (
	// DefaultDegree is the default B+ tree fanout.
	DefaultDegree = 12
	// DefaultMinFill is the default lower occupancy bound for internal balancing.
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
	// Degree is the max number of children for internal nodes.
	Degree int
	// MinFill is the minimum number of children for non-root internal nodes.
	MinFill int
	// Monoid aggregates summaries up the tree.
	Monoid SummaryMonoid[S]
}

func (cfg Config[S]) normalized() Config[S] {
	if cfg.Degree == 0 {
		cfg.Degree = DefaultDegree
	}
	if cfg.MinFill == 0 {
		cfg.MinFill = cfg.Degree / 2
	}
	return cfg
}

func (cfg Config[S]) validate() error {
	cfg = cfg.normalized()
	if cfg.Monoid == nil {
		return fmt.Errorf("%w: monoid is required", ErrInvalidConfig)
	}
	if cfg.Degree < 4 {
		return fmt.Errorf("%w: degree must be >= 4", ErrInvalidConfig)
	}
	if cfg.MinFill < 2 || cfg.MinFill > cfg.Degree/2 {
		return fmt.Errorf("%w: minFill must be in [2, degree/2]", ErrInvalidConfig)
	}
	if err := validateBackendConfig(cfg); err != nil {
		return err
	}
	return nil
}
