//go:build !cgo

package main

import (
	"fmt"
)

// openDuckDB is the CGo-disabled stand-in. DuckDB requires CGo (it
// statically links a C++ engine), so openBackend rejects the backend before
// this can run (see backendSpec.needsCGo).
func openDuckDB(openEnv) (binding, error) {
	return binding{}, fmt.Errorf("%w: duckdb requires CGo", errBackendUnavailable)
}
