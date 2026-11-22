package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/mysql/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

// compareWithDiskPaths runs the benchmark against each backend, setting
// per-backend DiskPath so disk metrics are populated in comparison tables.
// This mirrors benchkit.Compare but injects the correct DiskPath per backend.
// Backends are iterated in sorted order for deterministic progress output.
func compareWithDiskPaths(
	ctx context.Context,
	config benchkit.Config,
	factories map[string]benchkit.Factory,
	diskPaths map[string]string,
) map[string]*benchkit.Result {
	results := make(map[string]*benchkit.Result, len(factories))

	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}

	sort.Strings(names)

	for i, name := range names {
		factory := factories[name]
		cfg := config
		cfg.Backend = name
		cfg.DiskPath = diskPaths[name]

		if cfg.ProgressWriter != nil {
			fmt.Fprintf(cfg.ProgressWriter, "\n[%d/%d] backend: %s\n", i+1, len(names), name)
		}

		result, err := benchkit.Run(ctx, cfg, factory)
		if err != nil {
			results[name] = &benchkit.Result{
				Backend:   name,
				Profile:   cfg.Profile.Name,
				Timestamp: time.Now(),
				Error:     err.Error(),
			}

			continue
		}

		results[name] = result
	}

	return results
}

func makeFactory(
	ctx context.Context, backend, dsn, dir, durability string,
) (benchkit.Factory, string, func()) {
	var (
		diskPath string
		cleanup  func()
	)

	// Parse durability once, before any factory closure runs. This ensures
	// invalid input fails at CLI startup, not mid-benchmark.
	tier, tierSet, dErr := parseDurability(durability)
	if dErr != nil {
		fatalf("%v", dErr)
	}

	switch backend {
	case "system-memory", "sys-mem":
		return makeSystemMemoryFactory()

	case "memory", "mem":
		// Legacy path: the stack/memory preset bundle. The presets are
		// v5-doomed (ADR-0123) and kept for backend coverage the system
		// path has not grown yet; the system-backed entry point is
		// "system-memory" above. `memory` deliberately stays on the preset
		// so historical benchmark numbers remain comparable (the system
		// adapter honestly skips the kv read-model phases).
		return makeMemoryFactory(ctx)

	case "sqlite", "sq", "sqlite-cgo", "sq3":
		return makeSQLiteFactory(ctx, backend, dsn, dir, tier, tierSet, &diskPath, &cleanup)

	case "bbolt", "bolt":
		return makeBBoltFactory(ctx, dir, &diskPath, &cleanup)

	case "pebble", "peb":
		return makePebbleFactory(ctx, dir, tier, tierSet, &diskPath, &cleanup)

	case "postgres", "pg":
		return makePostgresFactory(ctx, dsn, tier, tierSet)

	case "mysql", "maria", "mariadb":
		return makeMySQLFactory(ctx, dsn)

	case "turso":
		return makeTursoFactory(ctx, dir, tier, tierSet, &diskPath, &cleanup)

	case "duckdb", "duck":
		return duckdbFactory(ctx, dsn, dir)

	default:
		fatalf(
			"unknown backend: %s (use system-memory, memory, sqlite, sqlite-cgo, bbolt, pebble, postgres, mysql, duckdb, or turso)",
			backend,
		)

		return nil, "", nil // unreachable
	}
}

func makeSQLiteFactory(
	_ context.Context,
	backend, dsn, dir string,
	tier stack.DurabilityTier, tierSet bool,
	diskPath *string, cleanup *func(),
) (benchkit.Factory, string, func()) {
	driverName := "sqlite"

	if backend == "sqlite-cgo" || backend == "sq3" {
		if !sqliteCgoAvailable {
			fatalf(
				"sqlite-cgo backend requires CGo (mattn/go-sqlite3) — rebuild with CGO_ENABLED=1",
			)

			return nil, "", nil // unreachable
		}

		driverName = "sqlite3"
	}

	dbDir := dir
	if dbDir == "" {
		dbDir = mkTempDir()
		*cleanup = func() { _ = os.RemoveAll(dbDir) }
	}

	dbPath := filepath.Join(dbDir, "bench.db")
	if dsn == "" {
		dsn = dbPath
	}

	*diskPath = dbDir

	return func() (*stack.Bundle, error) { //nolint:contextcheck // stack.New does not accept a context
		opts := []sqlite.Option{
			// Enable production optimizations (64 MB cache, mmap, temp in
			// memory) so benchmarks reflect realistic deployment config,
			// not SQLite's conservative 2 MB default cache.
			sqlite.WithPragmas(sqlopt.WithOptimizations()),
		}

		if driverName != "sqlite" {
			opts = append(opts, sqlite.WithDriverName(driverName))
		}

		if tierSet {
			opts = append(opts, sqlite.WithDurability(tier))
		}

		return sqlite.New(dsn, opts...)
	}, *diskPath, *cleanup
}

func makeBBoltFactory(
	_ context.Context, dir string, diskPath *string, cleanup *func(),
) (benchkit.Factory, string, func()) {
	boltDir := dir
	if boltDir == "" {
		boltDir = mkTempDir()
		*cleanup = func() { _ = os.RemoveAll(boltDir) }
	}

	*diskPath = boltDir
	dbPath := filepath.Join(boltDir, "bench.db")

	return func() (*stack.Bundle, error) {
		return bbolt.New(dbPath)
	}, *diskPath, *cleanup
}

func makePebbleFactory(
	_ context.Context,
	dir string, tier stack.DurabilityTier, tierSet bool,
	diskPath *string, cleanup *func(),
) (benchkit.Factory, string, func()) {
	pebDir := dir
	if pebDir == "" {
		pebDir = mkTempDir()
		*cleanup = func() { _ = os.RemoveAll(pebDir) }
	}

	*diskPath = pebDir

	return func() (*stack.Bundle, error) {
		opts := []pebble.Option{}
		if tierSet {
			opts = append(opts, pebble.WithDurability(tier))
		}

		b, err := pebble.New(pebDir, opts...)
		if err != nil {
			return nil, err
		}

		return b.Bundle, nil
	}, *diskPath, *cleanup
}

func makePostgresFactory(
	ctx context.Context, dsn string, tier stack.DurabilityTier, tierSet bool,
) (benchkit.Factory, string, func()) {
	if dsn == "" {
		fatalf(
			"postgres backend requires --dsn (e.g. postgres://user:pass@localhost:5432/bench?sslmode=disable)",
		)
	}

	_ = ctx // factory closure is invoked by benchkit.Run with a fresh context

	return func() (*stack.Bundle, error) {
		opts := []postgres.Option{}
		if tierSet {
			opts = append(opts, postgres.WithDurability(tier))
		}

		return postgres.New(dsn, opts...)
	}, "", nil
}

func makeMySQLFactory(ctx context.Context, dsn string) (benchkit.Factory, string, func()) {
	if dsn == "" {
		fatalf(
			"mysql backend requires --dsn (e.g. user:password@tcp(localhost:3306)/bench?parseTime=true)",
		)
	}

	_ = ctx

	return func() (*stack.Bundle, error) {
		return mysql.New(dsn)
	}, "", nil
}

func makeTursoFactory(
	ctx context.Context,
	dir string, tier stack.DurabilityTier, tierSet bool,
	diskPath *string, cleanup *func(),
) (benchkit.Factory, string, func()) {
	dbDir := dir
	if dbDir == "" {
		dbDir = mkTempDir()
		*cleanup = func() { _ = os.RemoveAll(dbDir) }
	}

	*diskPath = dbDir
	dbPath := filepath.Join(dbDir, "bench.db")

	_ = ctx

	return func() (*stack.Bundle, error) { //nolint:contextcheck // stack.New does not accept a context
		opts := []turso.Option{}
		if tierSet {
			opts = append(opts, turso.WithDurability(tier))
		}

		b, err := turso.New(dbPath, opts...)
		if err != nil {
			return nil, err
		}

		return b.Bundle, nil
	}, *diskPath, *cleanup
}

func mkTempDir() string {
	dir, err := os.MkdirTemp("", "cqrs-bench-*")
	if err != nil {
		fatalf("create temp dir: %v", err)
	}

	return dir
}
