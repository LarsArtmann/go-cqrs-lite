//go:build cgo

package main

import (
	"context"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// duckdbFactory builds a DuckDB-backed bundle for benchmarking. DuckDB is an
// embedded analytical (columnar) engine — best for read-heavy, aggregation, and
// GROUP BY workloads. It requires CGo (statically links a C++ engine), hence the
// build tag: the rest of cqrs-bench stays pure-Go.
//
// An empty dsn uses an in-memory database; a file path persists to disk.
func duckdbFactory(_ context.Context, dsn, dir string) (benchkit.Factory, string, func()) {
	diskPath := dir

	if dsn == "" {
		// In-memory: no disk footprint.
		return func() (*stack.Bundle, error) { return duckdb.New("") }, diskPath, nil
	}

	if diskPath == "" {
		diskPath = filepath.Dir(dsn)
	}

	return func() (*stack.Bundle, error) { return duckdb.New(dsn) }, diskPath, nil //nolint:contextcheck // stack.New does not accept a context
}
