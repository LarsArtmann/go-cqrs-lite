package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/spf13/cobra"
)

// modules is the canonical list of library/SDK modules whose exported API
// surface is tracked by the stability gate. Every directory with a go.mod
// (except examples, integration, root workspace, and this tool itself) MUST
// appear here — TestEveryGoModDirIsInModulesList enforces this automatically.
//
//nolint:gochecknoglobals // package-level so the meta-test can verify coverage
var modules = []string{
	// Layer 0: leaf modules
	"id/v4",

	"id/v4/idtest",

	"dispatcher/v4",

	"kv/v4",

	"dedup/v4",

	"record/v4",

	// Layer 1
	"event/v4",

	"event/v4/eventtest",
	"command/v4",

	"command/v4/commandtest",

	"commandlifecycle/v4",

	"commandlifecycle/projections/v4",

	"query/v4",

	"query/v4/querytest",

	"idempotency/sqlstore/v4",

	// Layer 2
	"schema/v4",

	"snapshot/v4",

	"projection/v4",

	"deriver/v4",

	// Layer 3
	"decider/v4",

	"graph/v4",

	"scenario/v4",

	"projectionhost/v4",

	"queue/v4",

	"queue/sqlite/v4",

	"queue/postgres/v4",

	"queue/mysql/v4",

	"scheduling/v4",

	"scheduling/engine/v4",

	"scheduling/sqlstore/v4",

	"metadata/v4",

	"metaengine/v4",

	"metaengine/bench/v4",

	"metaengine/pebbleengine/v4",

	"metaengine/duckdbengine/v4",

	"metaengine/pgengine/v4",

	"metaengine/mysqlengine/v4",

	"metaengine/otelobserver/v4",

	"metaengine/irohengine/v4",

	"metaengine/irohengine/loopback/v4",

	"metaengine/irohengine/quic/v4",

	"metaengine/projectionadapter/v4",

	"metaengine/graphadapter/v4",

	"metaengine/sqliteengine/v4",

	"metaengine/bigtableengine/v4",

	"metaengine/tursoengine/v4",

	"metaengine/badgerengine/v4",

	"metaengine/bboltengine/v4",

	"metaengine/dgraphengine/v4",

	// Layer 4
	"claiming/v4",

	"storage/memory/v4",

	"signing/v4",

	"encryption/v4",

	"otel/v4",

	"otel/otlp/v4",

	// Layer 5
	"middleware/v4",

	"storage/v4",

	"storage/v4/sql",

	"storage/backuptest/v4",

	"storage/pebble/v4",

	"storage/bbolt/v4",

	"storage/turso/v4",

	"listing/v4",

	"watermill/v4",

	"prometheus/v4",

	// Composition (Bundle layer)
	"stack/v4",

	"stack/memory/v4",

	"stack/metaengine/v4",

	"stack/sqlite/v4",

	"stack/duckdb/v4",

	"stack/pebble/v4",

	"stack/bbolt/v4",

	"stack/postgres/v4",

	"stack/mysql/v4",

	"stack/turso/v4",

	"stack/bench/v4",

	// Tooling + catalog
	"testutil/v4",

	"testutil/mysqltestcontainer/v4",

	"testutil/pgtestcontainer/v4",

	"catalog/v4",

	"benchkit/v4",

	"core/v5",
	"system/v4",

	"system/integration/v4",

	"systemtest/v4",

	"systemscenario/v4",

	"systemcatalog/v4",

	"cmd/cqrs-lint/v4",

	"cmd/cqrs-bench/v4",

	"cmd/cqrs-upgrade/v4",

	"cmd/cqrs-gen/v4",

	"cmd/doc-check/v4",
}

type AppConfig struct {
	cmdguard.Config

	Update bool `default:"false" flag:"update" help:"Update the golden file instead of verifying"`
}

func main() {
	cli, err := cmdguard.NewCLI(
		"api-stability",
		"API surface stability checker for go-cqrs-lite",
		AppConfig{
			Config: cmdguard.Config{ //nolint:exhaustruct_v5 // defaults acceptable for this one-shot CLI
			},
			Update: false,
		},
		cmdguard.WithCLILong(
			"api-stability verifies the exported API surface of every go-cqrs-lite module against a golden file.",
		),
	)
	//art-dupl:accept cobra CLI bootstrap guard — identical by design across cmd tools
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "api-stability [--update]"
	rootCmd.RunE = func(_ *cobra.Command, _ []string) error {
		cfg := cli.Config()

		projectRoot := filepath.Join(".", "..", "..", "..")
		goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

		exports, err := collectAllModuleExports(modules, projectRoot)
		if err != nil {
			return err
		}

		sort.Strings(exports)

		// Nested module paths in the modules slice (e.g. "event" plus
		// "event/v4/eventtest") make the parent walk re-emit the child
		// module's symbols. The golden tracks a SET of exported symbols,
		// so collapse the duplicates before writing or comparing.
		exports = slices.Compact(exports)

		if cfg.Update {
			return writeGoldenFile(goldenPath, exports)
		}

		return verifyGoldenFile(goldenPath, exports)
	}

	cli.ExecuteAndExit(context.Background())
}

func collectAllModuleExports(modules []string, projectRoot string) ([]string, error) {
	var exports []string

	for _, mod := range modules {
		modPath := filepath.Join(projectRoot, mod)

		if _, err := os.Stat(modPath); err != nil {
			return nil, fmt.Errorf(
				"module %q not found at %s: %w — add it to the modules list or remove the stale entry",
				mod,
				modPath,
				err,
			)
		}

		exps, err := collectModuleExports(modPath)
		if err != nil {
			// Fail loudly: an unparseable module must never be mistaken for a
			// legitimately-shrinking golden (corruption looks identical to
			// removal otherwise).
			return nil, fmt.Errorf("module %s: %w", mod, err)
		}

		for _, e := range exps {
			exports = append(exports, mod+"/"+e)
		}
	}

	return exports, nil
}

var (
	errExportMismatch = errors.New("API export mismatch")
	errGoldenMissing  = errors.New("API golden file missing")
)

const (
	goldenDirPerms  = 0o750
	goldenFilePerms = 0o600
)

func writeGoldenFile(goldenPath string, exports []string) error {
	cleanPath := filepath.Clean(goldenPath)

	err := os.MkdirAll(filepath.Dir(cleanPath), goldenDirPerms)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	err = os.WriteFile(cleanPath, []byte(strings.Join(exports, "\n")+"\n"), goldenFilePerms)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if _, err := fmt.Fprintf(
		os.Stdout,
		"Updated %s (%d exports)\n",
		cleanPath,
		len(exports),
	); err != nil {
		return fmt.Errorf("write status: %w", err)
	}

	return nil
}

func verifyGoldenFile(goldenPath string, exports []string) error {
	cleanPath := filepath.Clean(goldenPath)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return handleReadError(cleanPath, err)
	}

	expected := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(exports) != len(expected) {
		return reportMismatch(expected, exports)
	}

	for i, exp := range expected {
		if exports[i] != exp {
			return fmt.Errorf(
				"%w: export %d: expected %q, got %q",
				errExportMismatch,
				i,
				exp,
				exports[i],
			)
		}
	}

	if _, err := fmt.Fprintf(
		os.Stdout,
		"API surface OK: %d exports verified\n",
		len(exports),
	); err != nil {
		return fmt.Errorf("write status: %w", err)
	}

	return nil
}

func handleReadError(goldenPath string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s; run with --update to create", errGoldenMissing, goldenPath)
	}

	return fmt.Errorf("read: %w", err)
}

func reportMismatch(expected, exports []string) error {
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

	return fmt.Errorf("%w: %d expected, %d actual", errExportMismatch, len(expected), len(exports))
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
