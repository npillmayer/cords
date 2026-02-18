//go:build btree_fixed

package btree

import "fmt"

func validateBackendConfig[S any](cfg Config[S]) error {
	if cfg.Degree > fixedMaxChildren {
		return fmt.Errorf("%w: degree must be <= %d for fixed backend", ErrInvalidConfig, fixedMaxChildren)
	}
	if cfg.MinFill > fixedBase {
		return fmt.Errorf("%w: minFill must be <= %d for fixed backend", ErrInvalidConfig, fixedBase)
	}
	return nil
}
