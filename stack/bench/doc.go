// Package bench benchmarks the [stack.Bundle] composition layer to prove
// zero-overhead: accessing stores through Bundle fields must be identical in
// ns/op and allocs/op to accessing the same stores directly.
//
// Run: go test -bench=. -benchmem ./...
package bench
