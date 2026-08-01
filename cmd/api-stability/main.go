package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// modules is the canonical list of library/SDK modules whose exported API
// surface is tracked by the stability gate. Every directory with a go.mod
// (except examples, integration, root workspace, and this tool itself) MUST
// appear here — TestEveryGoModDirIsInModulesList enforces this automatically.
//
//nolint:gochecknoglobals // package-level so the meta-test can verify coverage
var modules = []string{
	// Layer 0: leaf modules
	"id",
	"id/idtest",
	"dispatcher",
	"codec",
	"kv",
	"dedup",
	"retry",
	"flightrecorder",
	// Layer 1
	"event",
	"event/v4/eventtest",
	"command",
	"query",
	"query/querytest",
	"idempotency",
	"idempotency/kvstore",
	"idempotency/sqlstore",
	// Layer 2
	"schema",
	"snapshot",
	"projection",
	"deriver",
	// Layer 3
	"decider",
	"graph",
	"scenario",
	"projectionhost",
	"scheduling",
	"metadata",
	"metaengine",
	"metaengine/pebbleengine",
	"metaengine/duckdbengine",
	"metaengine/pgengine",
	"metaengine/projectionadapter",
	// Layer 4
	"storage/memory",
	"signing",
	"encryption",
	"otel",
	// Layer 5
	"middleware",
	"storage",
	"storage/sql",
	"storage/pebble",
	"storage/turso",
	"listing",
	"watermill",
	"prometheus",
	"transport/http",
	"transport/grpc",
	// Composition (Bundle layer)
	"stack",
	"stack/memory",
	"stack/sqlite",
	"stack/duckdb",
	"stack/pebble",
	"stack/postgres",
	"stack/mysql",
	"stack/turso",
	"stack/bench",
	// Tooling + catalog
	"testutil",
	"catalog",
	"benchkit",
	"cmd/cqrs-lint",
	"cmd/cqrs-bench",
	"cmd/cqrs-gen",
	"cmd/doc-check",
}

func main() {
	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	exports := collectAllModuleExports(modules, projectRoot)

	sort.Strings(exports)

	if len(os.Args) > 1 && os.Args[1] == "-update" {
		writeGoldenFile(goldenPath, exports)

		return
	}

	verifyGoldenFile(goldenPath, exports)
}

func collectAllModuleExports(modules []string, projectRoot string) []string {
	var exports []string

	for _, mod := range modules {
		modPath := filepath.Join(projectRoot, mod)

		if _, err := os.Stat(modPath); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: module %q not found at %s: %v\n", mod, modPath, err)
			fmt.Fprintf(os.Stderr, "Add it to the modules list or remove the stale entry.\n")
			os.Exit(1)
		}

		exps, err := collectExports(modPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", mod, err)

			continue
		}

		for _, e := range exps {
			exports = append(exports, mod+"/"+e)
		}
	}

	return exports
}

const (
	goldenDirPerms  = 0o750
	goldenFilePerms = 0o600
)

func writeGoldenFile(goldenPath string, exports []string) {
	cleanPath := filepath.Clean(goldenPath)

	err := os.MkdirAll(filepath.Dir(cleanPath), goldenDirPerms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(cleanPath, []byte(strings.Join(exports, "\n")+"\n"), goldenFilePerms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Updated %s (%d exports)\n", cleanPath, len(exports))

	os.Exit(0)
}

func verifyGoldenFile(goldenPath string, exports []string) {
	cleanPath := filepath.Clean(goldenPath)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		handleReadError(cleanPath, err)
	}

	expected := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(exports) != len(expected) {
		reportMismatch(expected, exports)
	}

	for i, exp := range expected {
		if exports[i] != exp {
			fmt.Fprintf(os.Stderr, "export %d: expected %q, got %q\n", i, exp, exports[i])
			os.Exit(1)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "API surface OK: %d exports verified\n", len(exports))
}

func handleReadError(goldenPath string, err error) {
	if os.IsNotExist(err) {
		fmt.Fprintf(
			os.Stderr,
			"golden file %s does not exist; run with -update to create\n",
			goldenPath,
		)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "read: %v\n", err)
	os.Exit(1)
}

func reportMismatch(expected, exports []string) {
	missing, added := diff(expected, exports)
	fmt.Fprintf(
		os.Stderr,
		"API surface mismatch: %d expected, %d actual\n",
		len(expected),
		len(exports),
	)

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "REMOVED exports:\n  %s\n", strings.Join(missing, "\n  "))
	}

	if len(added) > 0 {
		fmt.Fprintf(os.Stderr, "NEW exports:\n  %s\n", strings.Join(added, "\n  "))
	}

	os.Exit(1)
}

func diff(expected, actual []string) ([]string, []string) {
	expSet := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		expSet[e] = struct{}{}
	}

	actSet := make(map[string]struct{}, len(actual))
	for _, a := range actual {
		actSet[a] = struct{}{}
	}

	var missing []string

	for _, e := range expected {
		if _, ok := actSet[e]; !ok {
			missing = append(missing, e)
		}
	}

	var added []string

	for _, a := range actual {
		if _, ok := expSet[a]; !ok {
			added = append(added, a)
		}
	}

	return missing, added
}
