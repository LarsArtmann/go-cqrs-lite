package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
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

func makeFactory(backend, dsn, dir, durability string) (benchkit.Factory, string, func()) {
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
	case "memory", "mem":
		return func() (*stack.Bundle, error) { return memory.New() }, "", nil

	case "sqlite", "sq":
		dbDir := dir
		if dbDir == "" {
			dbDir = mkTempDir()
			cleanup = func() { _ = os.RemoveAll(dbDir) }
		}

		dbPath := filepath.Join(dbDir, "bench.db")
		if dsn == "" {
			dsn = dbPath
		}

		diskPath = dbDir

		return func() (*stack.Bundle, error) {
			opts := []sqlite.Option{}
			if tierSet {
				opts = append(opts, sqlite.WithDurability(tier))
			}

			return sqlite.New(dsn, opts...)
		}, diskPath, cleanup

	case "pebble", "peb":
		pebDir := dir
		if pebDir == "" {
			pebDir = mkTempDir()
			cleanup = func() { _ = os.RemoveAll(pebDir) }
		}

		diskPath = pebDir

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
		}, diskPath, cleanup

	case "postgres", "pg":
		if dsn == "" {
			fatalf(
				"postgres backend requires --dsn (e.g. postgres://user:pass@localhost:5432/bench?sslmode=disable)",
			)
		}

		return func() (*stack.Bundle, error) {
			opts := []postgres.Option{}
			if tierSet {
				opts = append(opts, postgres.WithDurability(tier))
			}

			return postgres.New(dsn, opts...)
		}, "", nil

	case "turso":
		dbDir := dir
		if dbDir == "" {
			dbDir = mkTempDir()
			cleanup = func() { _ = os.RemoveAll(dbDir) }
		}

		diskPath = dbDir
		dbPath := filepath.Join(dbDir, "bench.db")

		return func() (*stack.Bundle, error) {
			opts := []turso.Option{}
			if tierSet {
				opts = append(opts, turso.WithDurability(tier))
			}

			b, err := turso.New(dbPath, opts...)
			if err != nil {
				return nil, err
			}

			return b.Bundle, nil
		}, diskPath, cleanup

	case "duckdb", "duck":
		return duckdbFactory(dsn, dir)

	default:
		fatalf(
			"unknown backend: %s (use memory, sqlite, pebble, postgres, duckdb, or turso)",
			backend,
		)

		return nil, "", nil // unreachable
	}
}

func parseDurability(s string) (stack.DurabilityTier, bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strict":
		return stack.DurabilityStrict, true, nil
	case "normal":
		return stack.DurabilityNormal, true, nil
	case "relaxed":
		return stack.DurabilityRelaxed, true, nil
	case "":
		return stack.DurabilityNormal, false, nil
	default:
		return "", false, fmt.Errorf(
			"unknown durability tier: %s (use strict, normal, or relaxed)", s,
		)
	}
}

func mkTempDir() string {
	dir, err := os.MkdirTemp("", "cqrs-bench-*")
	if err != nil {
		fatalf("create temp dir: %v", err)
	}

	return dir
}

func parseCodec(name string) codec.Codec {
	switch name {
	case "json":
		return codec.JSONCodec{}
	case "cbor":
		return codec.CBORCodec{}
	default:
		fatalf("unknown codec: %s (use json or cbor)", name)

		return nil
	}
}

// parsePayloadSizes parses a comma-separated list of payload sizes (e.g.
// "64,256,4096") into an int slice. Returns nil for an empty string (meaning:
// use the single --payload-size). Returns an error on malformed input.
//
// pflag blindly consumes the next token as a string flag's value, so
// "--payload-sizes --profile stress" sets payload-sizes to "--profile". This
// guard detects that pattern and returns an actionable error instead of a
// confusing strconv error.
func parsePayloadSizes(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	if looksLikeFlag(s) {
		return nil, fmt.Errorf(
			"value %q looks like a flag name, not a size list — "+
				"use --flag=VALUE syntax (e.g. --payload-sizes=64,256,4096) "+
				"or place the flag last in the command line",
			s,
		)
	}

	parts := strings.Split(s, ",")
	sizes := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid size %q: %w", part, err)
		}

		if n <= 0 {
			return nil, errorfamily.Newf(errorfamily.Rejection,
				"cqrs-bench.invalid_size", "size must be > 0, got %d", n)
		}

		sizes = append(sizes, n)
	}

	if len(sizes) < 2 {
		return nil, errorfamily.Newf(errorfamily.Rejection,
			"cqrs-bench.too_few_sizes",
			"provide at least 2 sizes for a mixed workload, got %d", len(sizes))
	}

	return sizes, nil
}

// looksLikeFlag reports whether s appears to be a flag name that pflag
// consumed as a value (e.g. "--profile" or "-p"). Negative numbers like
// "-1" are NOT treated as flag-like.
func looksLikeFlag(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}

	// "--anything" is a long flag.
	if s[1] == '-' {
		return true
	}

	// "-x" where x is a letter is a short flag; "-1" or "-1.5" is not.
	return !strings.ContainsAny(s[1:2], "0123456789.")
}
