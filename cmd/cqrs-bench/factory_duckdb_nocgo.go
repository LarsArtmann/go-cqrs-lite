//go:build !cgo

package main

import "github.com/larsartmann/go-cqrs-lite/benchkit/v4"

// duckdbFactory is the CGo-disabled stub. DuckDB requires CGo (it statically
// links a C++ engine), so without CGo the backend cannot be used.
func duckdbFactory(_, _ string) (benchkit.Factory, string, func()) {
	fatalf("duckdb backend requires CGo — rebuild with CGO_ENABLED=1")

	return nil, "", nil // unreachable
}
