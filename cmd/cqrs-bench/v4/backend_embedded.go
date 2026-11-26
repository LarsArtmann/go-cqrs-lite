package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// openMemory benchmarks the stack/memory preset. It deliberately stays on
// the preset (the system-backed entry point is "system-memory") so historical
// numbers remain comparable.
func openMemory(openEnv) (binding, error) {
	return binding{
		name: "memory",
		factory: func() (*stack.Bundle, error) {
			bundle, err := memory.New()
			if err != nil {
				return nil, fmt.Errorf("open memory store: %w", err)
			}

			return bundle, nil
		},
	}, nil
}

// openSystemMemory crosses the benchkit bridge: the benchmark runs against a
// system.New deployment (the strategic composition root, ADR-0123) through
// benchkit.FactoryFromSystem instead of a stack preset. Capabilities the
// system does not expose (kv read models) skip with recorded warnings.
func openSystemMemory(openEnv) (binding, error) {
	deploy := func(ctx context.Context) (*system.System, error) {
		deployment := system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{
				"primary": {Driver: "memory"},
			},
			Instances: []system.InstanceConfig{
				{Role: system.RoleSourceOfTruth, Engine: "primary"},
				{Role: system.RoleProjections, Engine: "primary"},
			},
		}

		sys, err := system.New(ctx, system.DomainConfig{}, deployment)
		if err != nil {
			return nil, fmt.Errorf("boot system for benchmark: %w", err)
		}

		return sys, nil
	}

	return binding{name: "system-memory", factory: benchkit.FactoryFromSystem(deploy)}, nil
}

// openSQLite returns the opener for one SQLite driver: "" is the pure-Go
// modernc driver, "sqlite3" the CGo mattn driver.
func openSQLite(driver string) func(openEnv) (binding, error) {
	return func(env openEnv) (binding, error) {
		opts := []sqlite.Option{
			// Production optimizations (64 MB cache, mmap, temp in memory):
			// a benchmark should reflect a realistic deployment, not
			// SQLite's conservative 2 MB default cache.
			sqlite.WithPragmas(sqlopt.WithOptimizations()),
		}

		if driver != "" {
			opts = append(opts, sqlite.WithDriverName(driver))
		}

		if env.durabilitySet {
			opts = append(opts, sqlite.WithDurability(env.durability))
		}

		open := func(location string) (*stack.Bundle, error) {
			bundle, err := sqlite.New(location, opts...)
			if err != nil {
				return nil, fmt.Errorf("open sqlite store %s: %w", location, err)
			}

			return bundle, nil
		}

		if env.dsn != "" {
			return dsnBinding(env, "sqlite", "warmup.db", open)
		}

		site, err := env.site("sqlite", legacyFileStore)
		if err != nil {
			return binding{}, err
		}

		return dirBinding("sqlite", site, func(dir string) (*stack.Bundle, error) {
			return open(filepath.Join(dir, storeFileName))
		}), nil
	}
}

// openBBolt benchmarks the bbolt preset.
func openBBolt(env openEnv) (binding, error) {
	site, err := env.site("bbolt", legacyFileStore)
	if err != nil {
		return binding{}, err
	}

	return dirBinding("bbolt", site, func(dir string) (*stack.Bundle, error) {
		var opts []bbolt.Option
		if env.durabilitySet {
			opts = append(opts, bbolt.WithDurability(env.durability))
		}

		bundle, err := bbolt.New(filepath.Join(dir, storeFileName), opts...)
		if err != nil {
			return nil, fmt.Errorf("open bbolt store: %w", err)
		}

		return bundle, nil
	}), nil
}

// openPebble benchmarks the Pebble preset.
func openPebble(env openEnv) (binding, error) {
	site, err := env.site("pebble", legacyPebbleStore)
	if err != nil {
		return binding{}, err
	}

	return dirBinding("pebble", site, func(dir string) (*stack.Bundle, error) {
		var opts []pebble.Option
		if env.durabilitySet {
			opts = append(opts, pebble.WithDurability(env.durability))
		}

		opened, err := pebble.New(dir, opts...)
		if err != nil {
			return nil, fmt.Errorf("open pebble store: %w", err)
		}

		return opened.Bundle, nil
	}), nil
}

// openTurso benchmarks the Turso preset.
func openTurso(env openEnv) (binding, error) {
	site, err := env.site("turso", legacyFileStore)
	if err != nil {
		return binding{}, err
	}

	return dirBinding("turso", site, func(dir string) (*stack.Bundle, error) {
		var opts []turso.Option
		if env.durabilitySet {
			opts = append(opts, turso.WithDurability(env.durability))
		}

		opened, err := turso.New(filepath.Join(dir, storeFileName), opts...)
		if err != nil {
			return nil, fmt.Errorf("open turso store: %w", err)
		}

		return opened.Bundle, nil
	}), nil
}

// dirBinding prepares a backend whose stores are directories: the disk
// footprint is the measured store directory.
func dirBinding(backend string, site *storeSite, open func(dir string) (*stack.Bundle, error)) binding {
	return binding{
		name:     backend,
		factory:  site.factory(open),
		diskPath: site.measuredDir,
		site:     site,
	}
}

// dsnBinding prepares a backend whose store is addressed by --dsn. An
// in-memory DSN is a fresh store on every open. A file DSN names a store the
// operator owns: --replay reads it, and a measuring run requires it to be
// absent or empty and then replaces it for every generation (warmup writes
// to a throwaway file named warmupName in a private temp directory).
func dsnBinding(
	env openEnv, backend, warmupName string,
	open func(location string) (*stack.Bundle, error),
) (binding, error) {
	if isMemoryDSN(env.dsn) {
		return binding{
			name:    backend,
			factory: func() (*stack.Bundle, error) { return open(env.dsn) },
		}, nil
	}

	if !env.calls.replay {
		if err := requireEmptyFile(dsnFilePath(env.dsn)); err != nil {
			return binding{}, err
		}
	}

	site, err := newFileSite(env.dsn, warmupName, env.calls)
	if err != nil {
		return binding{}, err
	}

	return binding{name: backend, factory: site.factory(open), site: site}, nil
}

// isMemoryDSN reports whether a DSN describes a store that lives only in
// memory and is therefore empty on every open.
func isMemoryDSN(dsn string) bool {
	return dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:")
}

// dsnFilePath is the filesystem path behind a file DSN: the file: URI prefix
// and the query are dropped.
func dsnFilePath(dsn string) string {
	path, _, _ := strings.Cut(dsn, "?")
	path = strings.TrimPrefix(path, "file:")

	return filepath.Clean(path)
}

// legacyFileStore reports whether a --dir root holds the single database
// file that cqrs-bench wrote straight into --dir before stores moved under
// .cqrs-bench/<backend>/.
func legacyFileStore(root string) bool {
	_, err := os.Stat(filepath.Join(root, storeFileName))

	return err == nil
}

// legacyPebbleStore reports whether a --dir root is itself a Pebble store.
func legacyPebbleStore(root string) bool {
	manifests, err := filepath.Glob(filepath.Join(root, "MANIFEST-*"))

	return err == nil && len(manifests) > 0
}

// noLegacyStore marks backends that never wrote a store straight into --dir.
func noLegacyStore(string) bool {
	return false
}
