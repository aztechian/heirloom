package handlers

// ptr is a small test helper shared across this package's test files for
// constructing pointers to literals inline.
func ptr[T any](v T) *T { return &v }
