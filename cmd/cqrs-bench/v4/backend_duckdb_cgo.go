//go:build cgo

package main

import (
	"fmt"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// duckDBFileName is the database file name inside a tool-owned store
// directory.
const duckDBFileName = "bench.duckdb"

// openDuckDB benchmarks the DuckDB preset. DuckDB is an embedded analytical
// (columnar) engine, best for read-heavy, aggregation, and GROUP BY
// workloads. It requires CGo (a statically linked C++ engine), hence the
// build tag: the rest of cqrs-bench stays pure-Go.
//
// Without --dsn and --dir the store is in-memory (no disk footprint). A
// --dsn names the database file; --dir keeps one under the tool's subtree.
func openDuckDB(env openEnv) (binding, error) {
	open := func(location string) (*stack.Bundle, error) {
		bundle, err := duckdb.New(location)
		if err != nil {
			return nil, fmt.Errorf("open duckdb store %s: %w", location, err)
		}

		return bundle, nil
	}

	switch {
	case env.dsn != "":
		return dsnBinding(env, "duckdb", "warmup.duckdb", open)
	case env.dir == "":
		return binding{
			name:    "duckdb",
			factory: func() (*stack.Bundle, error) { return open("") },
		}, nil
	}

	site, err := env.site("duckdb", noLegacyStore)
	if err != nil {
		return binding{}, err
	}

	return dirBinding("duckdb", site, func(dir string) (*stack.Bundle, error) {
		return open(filepath.Join(dir, duckDBFileName))
	}), nil
}
