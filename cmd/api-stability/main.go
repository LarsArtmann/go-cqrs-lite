package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	"scheduling/sqlstore",
	"metadata",
	"metaengine",
	"metaengine/pebbleengine",
	"metaengine/duckdbengine",
	"metaengine/pgengine",
	"metaengine/irohengine",
	"metaengine/irohengine/loopback",
	"metaengine/irohengine/quic",
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
	"storage/bbolt",
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
	"stack/bbolt",
	"stack/postgres",
	"stack/mysql",
	"stack/turso",
	"stack/bench",
	// Tooling + catalog
	"testutil",
	"catalog",
	"benchkit",
	"system",
	"cmd/cqrs-lint",
	"cmd/cqrs-bench",
	"cmd/cqrs-gen",
	"cmd/doc-check",
}

type AppConfig struct {
	cmdguard.Config

	Update bool `default:"false" flag:"update" help:"Update the golden file instead of verifying"`
}

func main() {
	cli, err := cmdguard.NewCLI(
		"api-stability",
		"API surface stability checker for go-cqrs-lite",
		AppConfig{},
		cmdguard.WithCLILong(
			"api-stability verifies the exported API surface of every go-cqrs-lite module against a golden file.",
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CLI: %v\n", err)
		os.Exit(1)
	}

	rootCmd := cli.RootCommand()
	rootCmd.Use = "api-stability [--update]"
	rootCmd.RunE = func(_ *cobra.Command, _ []string) error {
		cfg := cli.Config()

		projectRoot := filepath.Join(".", "..", "..")
		goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

		exports, err := collectAllModuleExports(modules, projectRoot)
		if err != nil {
			return err
		}

		sort.Strings(exports)

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

		exps, err := collectExports(modPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", mod, err)

			continue
		}

		for _, e := range exps {
			exports = append(exports, mod+"/"+e)
		}
	}

	return exports, nil
}

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

	fmt.Fprintf(os.Stdout, "Updated %s (%d exports)\n", cleanPath, len(exports))

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
			return fmt.Errorf("export %d: expected %q, got %q", i, exp, exports[i])
		}
	}

	fmt.Fprintf(os.Stdout, "API surface OK: %d exports verified\n", len(exports))

	return nil
}

func handleReadError(goldenPath string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("golden file %s does not exist; run with --update to create", goldenPath)
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

	return fmt.Errorf("API surface mismatch: %d expected, %d actual", len(expected), len(exports))
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
