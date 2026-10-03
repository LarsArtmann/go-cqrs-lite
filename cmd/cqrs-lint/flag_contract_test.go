package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// This file pins the flag-level contract: vocabulary single-sourcing,
// acceptance/scoping matrices, boolean precedence, preset e2e, and
// flag-over-config precedence. (config parity, help golden, and command
// surfaces live in contract_enforcement_test.go.)

// TestCommandShortsDeriveFromVocabularies pins the single-sourcing: each
// multi-format command's Short advertises exactly its vocabulary slice, and
// the root --format help tag carries the full lint vocabulary.
func TestCommandShortsDeriveFromVocabularies(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)
	root := cli.RootCommand()

	cases := []struct {
		cmd     string
		formats []string
	}{
		{"doctor", formatsDoctor},
		{"scorecard", formatsScorecard},
		{"rules", formatsRules},
	}

	for _, tc := range cases {
		cmd, _, err := root.Find([]string{tc.cmd})
		if err != nil || cmd == nil || cmd.Name() != tc.cmd {
			t.Fatalf("find %s: %v", tc.cmd, err)
		}

		want := "formats: " + formatList(tc.formats)
		if !strings.Contains(cmd.Short, want) {
			t.Errorf("%s Short should advertise %q, got: %q", tc.cmd, want, cmd.Short)
		}
	}

	help := formatFlagHelpTag(t)
	for _, format := range formatsLint {
		if !strings.Contains(help, format) {
			t.Errorf("root --format help tag should list %q, got: %q", format, help)
		}
	}
}

// TestFormatVocabulariesAreSubsets pins the structural contract: every
// command vocabulary must be a subset of the lint vocabulary (the root
// validates the full set; a command advertising something the root rejects
// — or vice versa — is a split brain).
func TestFormatVocabulariesAreSubsets(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		formats []string
	}{
		{"scorecard", formatsScorecard},
		{"doctor", formatsDoctor},
		{"rules", formatsRules},
	} {
		for _, format := range tc.formats {
			if !slices.Contains(formatsLint, format) {
				t.Errorf(
					"%s vocabulary contains %q which the lint vocabulary rejects",
					tc.name, format,
				)
			}
		}
	}
}

// TestPersistentFlagAcceptanceMatrix pins the shared vocabulary: every
// subcommand ACCEPTS the persistent flags (format, min-severity,
// min-confidence, color, typed-info). Which of them each command actually
// CONSUMES is documented in the README flag-consumption matrix — cmdguard
// persistence is all-or-nothing, so acceptance is the mechanically
// enforceable half of that contract.
func TestPersistentFlagAcceptanceMatrix(t *testing.T) {
	t.Parallel()

	flagsWithValue := [][2]string{
		{"--format", "text"},
		{"--min-severity", "info"},
		{"--min-confidence", "low"},
		{"--color", "never"},
		{"--typed-info", "auto"},
	}

	for _, cmdName := range allSubcommands() {
		for _, fw := range flagsWithValue {
			t.Run(cmdName+" "+fw[0], func(t *testing.T) {
				t.Parallel()

				cli := newTestCLI(t)
				err := cli.ExecuteWithArgs(context.Background(), []string{
					cmdName, fw[0], fw[1], "--path", t.TempDir(),
				})
				if err != nil && strings.Contains(strings.ToLower(err.Error()), "unknown flag") {
					t.Errorf("%s %s: shared flag must be accepted, got: %v", cmdName, fw[0], err)
				}
			})
		}
	}
}

// TestLocalFlagsRejectedOnEverySubcommand pins the scoping half: lint-run
// flags without a subcommand counterpart are unknown-flag errors on EVERY
// subcommand. (--dry-run exists on doctor, --verbose on version, --json/
// --markdown on rules — those are command-local and excluded here.)
func TestLocalFlagsRejectedOnEverySubcommand(t *testing.T) {
	t.Parallel()

	localOnly := []string{
		"--fix", "--fast", "--only", "--exclude", "--exclude-rules", "--quiet",
		"--scorecard", "--fp-suspects", "--show-suppressed", "--strict-load",
		"--fail-on-stale-suppressions", "--adoption", "--health-score", "--group-by",
	}

	for _, cmdName := range allSubcommands() {
		for _, flag := range localOnly {
			t.Run(cmdName+" "+flag, func(t *testing.T) {
				t.Parallel()

				cli := newTestCLI(t)
				err := cli.ExecuteWithArgs(context.Background(), []string{cmdName, flag})
				if err == nil {
					t.Fatalf("%s %s: expected unknown-flag error, got nil", cmdName, flag)
				}
				if !strings.Contains(strings.ToLower(err.Error()), "unknown flag") {
					t.Errorf("%s %s: expected unknown-flag error, got: %v", cmdName, flag, err)
				}
			})
		}
	}
}

// TestRulesBooleanFlagPrecedence pins the documented precedence: when rules
// gets both --json and --markdown, markdown wins (byte-compatible with the
// pre-shared-format behavior where markdown printed last).
func TestRulesBooleanFlagPrecedence(t *testing.T) {
	t.Parallel()

	format, err := rulesFormat(rulesFlags{JSON: true, Markdown: true}, &AppConfig{})
	if err != nil {
		t.Fatalf("both booleans set: unexpected error: %v", err)
	}
	if format != "markdown" {
		t.Errorf("rules --json --markdown: markdown must win, got %q", format)
	}
}

// TestInitPresetE2e drives `init --preset X` through the full CLI for every
// named preset and round-trips the written file through the REAL config
// loader (JSONCLoader), not just raw json.Unmarshal: the exact parse path
// production uses must accept every template the command can write.
func TestInitPresetE2e(t *testing.T) {
	t.Parallel()

	for _, name := range presetNamesForTest(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			cli := newTestCLI(t)

			if err := cli.ExecuteWithArgs(
				context.Background(), []string{"init", "--preset", name, "--path", dir},
			); err != nil {
				t.Fatalf("init --preset %s: %v", name, err)
			}

			data, err := os.ReadFile(filepath.Join(dir, ".cqrs-lint.json"))
			if err != nil {
				t.Fatalf("preset config not written: %v", err)
			}

			var cfg AppConfig
			if _, err := (JSONCLoader{}).Load(data, &cfg); err != nil {
				t.Fatalf("preset %q config rejected by JSONCLoader: %v\nconfig:\n%s", name, err, data)
			}
			if string(cfg.Preset) != name {
				t.Errorf("preset %q: loaded config carries %q", name, cfg.Preset)
			}
		})
	}
}

// TestFormatFlagBeatsConfigFile pins cmdguard's precedence: an explicit
// --format flag wins over a config-file value. The config says text; the
// flag says json; scorecard must emit JSON.
func TestFormatFlagBeatsConfigFile(t *testing.T) {
	dir := tempDirWithConfig(t, `{"format": "text"}`)
	t.Chdir(dir)

	cli := newTestCLI(t)

	out := captureStdout(t, func() {
		err := cli.ExecuteWithArgs(context.Background(), []string{
			"scorecard", "--format", "json", "--path", dir,
		})
		if err != nil {
			t.Errorf("scorecard --format json over config text: %v", err)
		}
	})

	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "{") || !json.Valid([]byte(trimmed)) {
		t.Errorf("--format flag must beat config file value, got: %.120s", trimmed)
	}
}
