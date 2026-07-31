package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// compareWithDiskPaths runs the benchmark against each backend, setting
// per-backend DiskPath so disk metrics are populated in comparison tables.
// This mirrors benchkit.Compare but injects the correct DiskPath per backend.
func compareWithDiskPaths(
	ctx context.Context,
	config benchkit.Config,
	factories map[string]benchkit.Factory,
	diskPaths map[string]string,
) map[string]*benchkit.Result {
	results := make(map[string]*benchkit.Result, len(factories))

	for name, factory := range factories {
		cfg := config
		cfg.Backend = name
		cfg.DiskPath = diskPaths[name]

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
			if tier, ok := parseDurability(durability); ok {
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
			if tier, ok := parseDurability(durability); ok {
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
			if tier, ok := parseDurability(durability); ok {
				opts = append(opts, postgres.WithDurability(tier))
			}
			return postgres.New(dsn, opts...)
		}, "", nil

	case "duckdb", "duck":
		return duckdbFactory(dsn, dir)

	default:
		fatalf("unknown backend: %s (use memory, sqlite, pebble, postgres, duckdb, or turso)", backend)

		return nil, "", nil // unreachable
	}
}

func parseDurability(s string) (stack.DurabilityTier, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strict":
		return stack.DurabilityStrict, true
	case "normal":
		return stack.DurabilityNormal, true
	case "relaxed":
		return stack.DurabilityRelaxed, true
	case "":
		return stack.DurabilityNormal, false
	default:
		fatalf("unknown durability tier: %s (use strict, normal, or relaxed)", s)
		return "", false
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
func parsePayloadSizes(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
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
