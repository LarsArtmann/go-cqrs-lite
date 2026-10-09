package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// backendRowKeys maps a detected StoreKind to the catalog row that tracks
// that backend. Kinds without a catalog row (badger, dgraph, iroh, bigtable,
// custom) have no stack-preset entry and are skipped.
var backendRowKeys = map[analyzer.StoreKind]string{
	analyzer.StoreSQLite:   "stack/sqlite",
	analyzer.StorePostgres: "stack/postgres",
	analyzer.StoreMySQL:    "stack/mysql",
	analyzer.StorePebble:   "stack/pebble",
	analyzer.StoreBolt:     "stack/bbolt",
	analyzer.StoreTurso:    "stack/turso",
	analyzer.StoreDuckDB:   "stack/duckdb",
	analyzer.StoreMemory:   "stack/memory",
}

// engineNames maps a StoreKind to the metaengine driver name that provides
// it, for evidence attribution. The memory engine is built into metaengine
// core (registered without an engine import), so it gets its own evidence
// wording.
var engineNames = map[analyzer.StoreKind]string{
	analyzer.StoreSQLite:   "sqlite",
	analyzer.StorePostgres: "postgres",
	analyzer.StoreMySQL:    "mysql",
	analyzer.StorePebble:   "pebble",
	analyzer.StoreBolt:     "bbolt",
	analyzer.StoreTurso:    "turso",
	analyzer.StoreDuckDB:   "duckdb",
	analyzer.StoreMemory:   "memory",
}

// creditComposition upgrades MISSING persistence rows to Used when the
// FeatureProfile proves the backend is wired through composition — a
// metaengine engine import, a raw database driver import, or system.New —
// instead of a direct stack-preset import. Without this, a system.New
// consumer reads "SQLite Stack: MISSING" while running SQLite, and one lying
// row licenses dismissing every row.
//
// Credit never overrides a direct import (the import evidence is stronger)
// and never pulls profile-Irrelevant rows into the score.
func creditComposition(
	usage map[analyzer.ModuleKey]analyzer.ModuleUsage,
	fp analyzer.FeatureProfile,
	relevantSet map[analyzer.ModuleKey]bool,
) map[analyzer.ModuleKey]analyzer.ModuleUsage {
	credited := maps.Clone(usage)
	for _, kind := range fp.EffectiveStores() {
		key, ok := backendRowKeys[kind]
		if !ok {
			continue
		}
		upgradeComposed(
			credited, analyzer.ModuleKey(key),
			evidenceForBackend(kind, fp), relevantSet,
		)
	}
	creditSystemRows(credited, fp, relevantSet)
	return credited
}

// creditSystemRows credits the rows every system.New composition provides
// regardless of engine choice: system always embeds the in-memory kernel
// (storage/memory) and wires the storage/ layer internally.
func creditSystemRows(
	credited map[analyzer.ModuleKey]analyzer.ModuleUsage,
	fp analyzer.FeatureProfile,
	relevantSet map[analyzer.ModuleKey]bool,
) {
	if !fp.HasSystemComposition {
		return
	}
	upgradeComposed(
		credited, "storage",
		"composed via system.New (system wires storage/ internally)", relevantSet,
	)
	upgradeComposed(
		credited, "stack/memory",
		"composed via system.New (in-memory kernel embedded)", relevantSet,
	)
}

// upgradeComposed raises an absent row to UsageImported with composition
// evidence. Direct imports win (their evidence names the import path) and
// irrelevant rows are never credited.
func upgradeComposed(
	credited map[analyzer.ModuleKey]analyzer.ModuleUsage,
	key analyzer.ModuleKey,
	evidence string,
	relevantSet map[analyzer.ModuleKey]bool,
) {
	if !relevantSet[key] || credited[key].Status >= analyzer.UsageImported {
		return
	}
	credited[key] = analyzer.ModuleUsage{
		Key: key, Status: analyzer.UsageImported, Evidence: evidence,
	}
}

// evidenceForBackend names every composition signal that proves the backend
// is in use, most specific first. An unattributed signal (a raw driver
// import) still credits the row, with generic wording.
func evidenceForBackend(kind analyzer.StoreKind, fp analyzer.FeatureProfile) string {
	var parts []string
	if engine, ok := engineNames[kind]; ok && slices.Contains(fp.MetaengineEngines, engine) {
		if engine == "memory" {
			parts = append(parts, "composed via metaengine built-in memory driver")
		} else {
			parts = append(parts, fmt.Sprintf("composed via metaengine %s engine import", engine))
		}
	}
	if fp.HasSystemComposition {
		parts = append(parts, "composed via system.New (wired internally)")
	}
	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("composed: %s backend detected (store signal)", kind))
	}
	return strings.Join(parts, "; ")
}
