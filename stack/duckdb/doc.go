// Package duckdb provides a DuckDB-backed preset for [stack.Bundle].
//
// DuckDB is an embedded analytical (OLAP) SQL engine. It excels at read-heavy
// analytical workloads, complex aggregations, window functions, and columnar
// scans. For write-heavy OLTP workloads, consider the sqlite or postgres presets.
//
// # Quick Start
//
//	b, err := duckdb.New("analytics.db")
//	defer b.Close()
//
// In-memory (ephemeral):
//
//	b, err := duckdb.New("")
//	defer b.Close()
//
// # CGo Requirement
//
// DuckDB statically links a C++ engine (~30-50MB binary increase). This module
// requires CGO_ENABLED=1 and a C/C++ compiler (gcc or clang). It is isolated
// in its own Go module so that consumers who do not import it never need CGo.
//
// The rest of go-cqrs-lite remains pure-Go (modernc.org/sqlite, pebble). Only
// code that imports stack/duckdb pulls in the CGo dependency.
//
// # Performance Tuning
//
// Limit threads and memory for resource-constrained environments:
//
//	b, err := duckdb.New("analytics.db",
//	    duckdb.WithThreads(4),
//	    duckdb.WithMemoryLimit("1GB"),
//	)
//
// # Multi-Database Topology
//
// Split concerns across separate database files:
//
//	b, err := duckdb.New("primary.db",
//	    duckdb.WithDSN(
//	        sqlopt.WithEventDB("events.db"),
//	        sqlopt.WithQueryDB("queries.db"),
//	        sqlopt.WithViewDB("views.db"),
//	    ),
//	)
//
// # Analytical Read Models
//
// DuckDB's columnar engine makes [SQLViewModel] tables especially powerful for
// dashboard queries, GROUP BY aggregations, and analytical projections. Use the
// DuckDB preset when your read model workload is dominated by analytical scans
// rather than point lookups.
//
// New runs schema migration by default. Disable with [WithDSN] (passing
// [sqlopt.WithoutAutoMigrate]).
package duckdb
