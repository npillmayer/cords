//go:build !btree_fixed

package btree

func validateBackendConfig[S any](cfg Config[S]) error {
	return nil
}
