package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// backendSpec declares one benchmarkable backend: its command-line names,
// the flags it honors, and how its stores are opened. The registry is the
// single source of truth for the backend vocabulary: validation, alias
// resolution, and the long description all derive from it.
type backendSpec struct {
	name    string
	aliases []string
	// summary is the one-line description shown in help.
	summary string
	// dsnExample documents the --dsn shape; empty when the backend has none.
	dsnExample string
	// needsDSN refuses to start without --dsn (database servers).
	needsDSN bool
	// needsCGo: the backend exists only in a CGo build.
	needsCGo bool
	// honorsDSN, honorsDir, and honorsTier say whether --dsn, --dir, and
	// --durability change where or how the store is written.
	honorsDSN  bool
	honorsDir  bool
	honorsTier bool
	// open prepares the backend's stores.
	open func(openEnv) (binding, error)
}

// backendRegistry lists every backend in help order.
func backendRegistry() []backendSpec {
	return []backendSpec{
		{
			name: "memory", aliases: []string{"mem"},
			summary: "In-memory store (stack/memory preset; no persistence)",
			open:    openMemory,
		},
		{
			name: "system-memory", aliases: []string{"sys-mem"},
			summary: "In-memory store behind the system.New composition root (ADR-0123); " +
				"phases the system does not expose are skipped",
			open: openSystemMemory,
		},
		{
			name: "sqlite", aliases: []string{"sq"},
			summary:   "SQLite database (pure-Go modernc driver, optimized pragmas)",
			honorsDSN: true, honorsDir: true, honorsTier: true,
			open: openSQLite(""),
		},
		{
			name: "sqlite-cgo", aliases: []string{"sq3"},
			summary:  "SQLite database (CGo mattn driver, 3-5x faster, requires gcc)",
			needsCGo: true, honorsDSN: true, honorsDir: true, honorsTier: true,
			open: openSQLite("sqlite3"),
		},
		{
			name: "bbolt", aliases: []string{"bolt"},
			summary:   "B+tree store (pure Go, single-writer, etcd-backed)",
			honorsDir: true, honorsTier: true,
			open: openBBolt,
		},
		{
			name: "pebble", aliases: []string{"peb"},
			summary:   "PebbleDB LSM-tree store",
			honorsDir: true, honorsTier: true,
			open: openPebble,
		},
		{
			name:      "turso",
			summary:   "Turso embedded database (libSQL/SQLite fork)",
			honorsDir: true, honorsTier: true,
			open: openTurso,
		},
		{
			name: "duckdb", aliases: []string{"duck"},
			summary:  "DuckDB columnar analytical engine (requires CGo)",
			needsCGo: true, honorsDSN: true, honorsDir: true,
			open: openDuckDB,
		},
		{
			name: "postgres", aliases: []string{"pg"},
			summary:    "PostgreSQL database server",
			dsnExample: "postgres://user:pass@localhost:5432/bench?sslmode=disable",
			needsDSN:   true, honorsDSN: true, honorsTier: true,
			open: openPostgres,
		},
		{
			name: "mysql", aliases: []string{"maria", "mariadb"},
			summary:    "MySQL/MariaDB database server",
			dsnExample: "user:password@tcp(localhost:3306)/bench?parseTime=true",
			needsDSN:   true, honorsDSN: true,
			open: openMySQL,
		},
	}
}

// lookupBackend resolves a command-line backend name, aliases included.
func lookupBackend(name string) (backendSpec, bool) {
	for _, spec := range backendRegistry() {
		if spec.name == name || slices.Contains(spec.aliases, name) {
			return spec, true
		}
	}

	var unknown backendSpec

	return unknown, false
}

// backendNames lists the canonical backend names in help order.
func backendNames() []string {
	return backendSummaries(func(spec backendSpec) string { return spec.name })
}

// backendSummaries maps the registry in help order.
func backendSummaries(pick func(backendSpec) string) []string {
	names := make([]string, 0, len(backendRegistry()))

	for _, spec := range backendRegistry() {
		names = append(names, pick(spec))
	}

	return names
}

// backendHelp is the "Backends:" block of the long description: one line
// per backend, aliases included.
func backendHelp() string {
	lines := []string{"Backends:"}

	for _, spec := range backendRegistry() {
		line := fmt.Sprintf("  %-13s %s", spec.name, spec.summary)
		if len(spec.aliases) > 0 {
			line += " (aliases: " + strings.Join(spec.aliases, ", ") + ")"
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// openEnv carries the operator's choices into a backend opener.
type openEnv struct {
	dsn           string
	dir           string
	durability    stack.DurabilityTier
	durabilitySet bool
	calls         storeCalls
	// generations is set when the command opens more than one measured
	// store (repeats, a soak): backends that cannot isolate theirs warn.
	generations bool
	cgo         bool
}

// site returns where a directory-backed backend keeps its stores: the store
// an earlier run left under --dir (--replay), the tool's subtree under
// --dir, or a private temp site. legacy reports whether a --dir root itself
// is a store that an older cqrs-bench wrote straight into it.
func (e openEnv) site(backend string, legacy func(root string) bool) (*storeSite, error) {
	switch {
	case e.calls.replay:
		if e.dir == "" {
			return nil, fmt.Errorf("%w: --replay needs --dir naming the store to read", errFlagCombo)
		}

		return newReplaySite(e.dir, backend, legacy)
	case e.dir == "":
		return newTempSite(e.calls)
	default:
		return newDirSite(e.dir, backend, e.calls)
	}
}

// binding is a prepared backend: benchkit's factory, the directory whose
// size is the run's disk footprint, and the site to clean up afterwards.
type binding struct {
	name     string
	factory  benchkit.Factory
	diskPath string
	site     *storeSite
	// warnings tell the operator about flags the backend ignores or limits
	// it cannot overcome.
	warnings []string
}

// close removes the scratch the binding owns.
func (b binding) close() error {
	if b.site == nil {
		return nil
	}

	return b.site.close()
}

// openBackend prepares the named backend for env. Every requirement is
// checked before a store is created, so an unsatisfiable command fails
// before it touches the disk.
func openBackend(name string, env openEnv) (binding, error) {
	spec, ok := lookupBackend(name)
	if !ok {
		return binding{}, fmt.Errorf("%w %q (use: %s)", errBackend, name, strings.Join(backendNames(), ", "))
	}

	if spec.needsCGo && !env.cgo {
		return binding{}, fmt.Errorf("%w: %q requires CGo, rebuild with CGO_ENABLED=1",
			errBackendUnavailable, spec.name)
	}

	if spec.needsDSN && env.dsn == "" {
		return binding{}, fmt.Errorf("%w: backend %q needs a connection string, e.g. %s",
			errDSNRequired, spec.name, spec.dsnExample)
	}

	bound, err := spec.open(env)
	if err != nil {
		return binding{}, fmt.Errorf("backend %q: %w", spec.name, err)
	}

	bound.name = spec.name
	bound.warnings = append(bound.warnings, spec.ignoredFlags(env)...)

	return bound, nil
}

// ignoredFlags names the flags the backend silently does not act on.
func (s backendSpec) ignoredFlags(env openEnv) []string {
	var ignored []string

	if env.dsn != "" && !s.honorsDSN {
		ignored = append(ignored, "--dsn")
	}

	if env.dir != "" && !s.honorsDir {
		ignored = append(ignored, "--dir")
	}

	if env.durabilitySet && !s.honorsTier {
		ignored = append(ignored, "--durability")
	}

	return ignored
}
