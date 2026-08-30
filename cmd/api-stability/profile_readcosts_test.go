package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEngineProfilesSetReadCosts is a source-scan meta-test pinning that
// every persistent engine's cost profile sets the 4-field ReadCosts block.
// The core metaengine module cannot import the engine modules (dependency
// isolation), so — like TestEveryGoModDirIsInModulesList — this test reads
// the repo tree instead of constructing engines.
//
// Roster: the 8 persistent engines. Memory (volatile test-tier) and the
// iroh passthrough (inherits the local engine's profile) are exempt with
// recorded reasons. A new engine module must be added to the roster here in
// the same commit that adds the engine.
func TestEngineProfilesSetReadCosts(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")

	// engine dir → file carrying its ReadCosts block (profile factory or
	// Profile method).
	roster := map[string]string{
		"metaengine/pgengine":     "engine.go",
		"metaengine/mysqlengine":  "engine.go",
		"metaengine/sqliteengine": "../engine.go", // profile factory lives core-side (SQLiteEngineProfile)
		"metaengine/duckdbengine": "engine.go",
		"metaengine/dgraphengine": "engine.go",
		"metaengine/badgerengine": "engine.go",
		"metaengine/bboltengine":  "engine.go",
		"metaengine/pebbleengine": "engine.go",
	}

	// Engines intentionally without ReadCosts, with the reason recorded here
	// so the exemption is a decision, not an omission.
	exempt := map[string]string{
		"metaengine":             "core module — memory engine is volatile/test-tier; scalar NsPerOp fallback is intentional",
		"metaengine/tursoengine": "remote HTTP/RTT-dominated; per-row costs pending live calibration window",
		"metaengine/irohengine":  "replication passthrough — delegates reads to the local engine's profile",
	}

	for dir, file := range roster {
		path := filepath.Join(projectRoot, dir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("engine %s: read %s: %v", dir, path, err)
			continue
		}

		if !strings.Contains(string(data), "ReadCosts:") {
			t.Errorf("engine %s: no ReadCosts block found in %s — every persistent engine must set per-pattern ReadCosts (ADR-0133; see the badger/bbolt/pebble 2026-08-30 wave for the bench pattern)", dir, file)
		}
	}

	for dir, reason := range exempt {
		if _, ok := roster[dir]; ok {
			t.Errorf("engine %s listed in both roster and exempt: %s", dir, reason)
		}
	}
}
