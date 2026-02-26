// Package conv provides small type-conversion helpers shared across packages.
package conv

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}
