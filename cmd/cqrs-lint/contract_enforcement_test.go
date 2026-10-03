package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// This file mechanically enforces the shipped CLI contract claims that
// subcommand_consistency_test.go does not cover:
//
//  1. Config-file format parity: a .cqrs-lint.json setting "format": "json"
//     switches EVERY multi-format command to JSON, not just the lint run
//     (the README claims this; these tests make it true or fail).
//  2. Help-drift golden: every subcommand registered on the CLI appears in
//     the hand-written usage block of the root long help, and vice versa —
//     the `changelog` omission class can never recur silently.
//  3. Surface tests for cobra's auto-added `completion` and `help [cmd]`
//     under the local-flag scoping.
//  4. Format vocabularies are single-sourced: every command's Short string
//     and the root --format help tag derive from the formats* slices.

// tempDirWithConfig creates a temp dir containing a .cqrs-lint.json with the
// given content and returns the dir path. Config-parity tests chdir into it
// because cmdguard discovers config files relative to the working directory.
func tempDirWithConfig(t *testing.T, configJSON string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, ".cqrs-lint.json"),
		[]byte(configJSON),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	return dir
}

// TestScorecardConfigFileFormatParity pins that the scorecard subcommand
// honors the config file's "format" key exactly like an explicit --format
// flag — the README's config-parity claim for multi-format commands.
func TestScorecardConfigFileFormatParity(t *testing.T) {
	dir := tempDirWithConfig(t, `{"format": "json"}`)
	t.Chdir(dir)

	cli := newTestCLI(t)

	out := captureStdout(t, func() {
		err := cli.ExecuteWithArgs(context.Background(), []string{"scorecard", "--path", dir})
		if err != nil {
			t.Errorf("scorecard with config format json: %v", err)
		}
	})

	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "{") {
		t.Errorf("config file format json should make scorecard emit JSON, got: %.120s", trimmed)
	}
	if !json.Valid([]byte(trimmed)) {
		t.Errorf("scorecard output should be valid JSON, got: %.120s", trimmed)
	}
}

// TestDoctorConfigFileFormatParity is the doctor twin of the scorecard
// parity test.
func TestDoctorConfigFileFormatParity(t *testing.T) {
	dir := tempDirWithConfig(t, `{"format": "json"}`)
	t.Chdir(dir)

	cli := newTestCLI(t)

	out := captureStdout(t, func() {
		err := cli.ExecuteWithArgs(context.Background(), []string{"doctor", "--path", dir})
		if err != nil {
			t.Errorf("doctor with config format json: %v", err)
		}
	})

	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "{") {
		t.Errorf("config file format json should make doctor emit JSON, got: %.120s", trimmed)
	}
	if !json.Valid([]byte(trimmed)) {
		t.Errorf("doctor output should be valid JSON, got: %.120s", trimmed)
	}
}

// documentedCommands extracts the subcommand names from the hand-written
// "Usage:" block of the root long help (`  cqrs-lint rules ...`).
func documentedCommands(t *testing.T, long string) map[string]string {
	t.Helper()

	documented := map[string]string{}
	re := regexp.MustCompile(`(?m)^  cqrs-lint ([a-z]+)\s+(.+)$`)

	for _, match := range re.FindAllStringSubmatch(long, -1) {
		documented[match[1]] = strings.TrimSpace(match[2])
	}

	if len(documented) == 0 {
		t.Fatal("no documented commands found in root long help — usage block missing?")
	}

	return documented
}

// registeredCommands returns the CLI's non-hidden subcommands, excluding
// cobra's auto-added help/completion commands (they get their own surface
// tests below).
func registeredCommands(root *cobra.Command) []string {
	var names []string

	for _, cmd := range root.Commands() {
		if cmd.Hidden {
			continue
		}

		switch cmd.Name() {
		case "help", "completion":
			continue
		}

		names = append(names, cmd.Name())
	}

	return names
}

// TestHelpListsEveryRegisteredCommand is the help-drift golden: the
// hand-written usage block and the actually-registered subcommands must be
// the same set. `changelog` was once added without updating the block — this
// test turns that class into a mechanical failure.
func TestHelpListsEveryRegisteredCommand(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)
	root := cli.RootCommand()

	documented := documentedCommands(t, rootLongHelp)
	for _, name := range registeredCommands(root) {
		if _, ok := documented[name]; !ok {
			t.Errorf("subcommand %q is registered but missing from the root usage block", name)
		}
	}

	for name := range documented {
		cmd, _, err := root.Find([]string{name})
		if err != nil || cmd == nil || cmd.Name() != name {
			t.Errorf("usage block documents %q but no such subcommand is registered", name)
		}
	}
}

// TestCompletionBashSucceeds pins that cobra's auto-added completion command
// still executes under the local-flag scoping (an unknown-flag regression
// here would break shell setup for every user).
func TestCompletionBashSucceeds(t *testing.T) {
	cli := newTestCLI(t)

	out := captureStdout(t, func() {
		if err := cli.ExecuteWithArgs(
			context.Background(),
			[]string{"completion", "bash"},
		); err != nil {
			t.Errorf("completion bash: %v", err)
		}
	})

	if !strings.Contains(out, "bash") || len(out) < 100 {
		t.Errorf("completion bash should emit a shell script, got %d bytes", len(out))
	}
}

// TestHelpSubcommandSurface pins `help <cmd>` (cobra's auto-added help
// command) and that it surfaces doctor's renamed prune flag.
func TestHelpSubcommandSurface(t *testing.T) {
	cli := newTestCLI(t)

	out := captureStdout(t, func() {
		if err := cli.ExecuteWithArgs(
			context.Background(),
			[]string{"help", "doctor"},
		); err != nil {
			t.Errorf("help doctor: %v", err)
		}
	})

	for _, want := range []string{"prune-suppressions", "audit-suppressions", "dry-run"} {
		if !strings.Contains(out, want) {
			t.Errorf("help doctor should mention %q, got: %.200s", want, out)
		}
	}
}

// TestCommandShortsDeriveFromVocabularies pins M02's single-sourcing: each
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

	field, ok := reflect.TypeOf(AppConfig{}).FieldByName("Format")
	if !ok {
		t.Fatal("AppConfig.Format field not found")
	}

	help := field.Tag.Get("help")
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
					tc.name,
					format,
				)
			}
		}
	}
}

// TestPersistentFlagAcceptanceMatrix pins the shared vocabulary: every
// subcommand ACCEPTS the persistent flags (path, format, min-severity,
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

	for _, cmdName := range []string{
		"rules", "version", "init", "doctor", "scorecard", "changelog", "explain",
	} {
		for _, fw := range flagsWithValue {
			cmdName, fw := cmdName, fw
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

	for _, cmdName := range []string{
		"rules", "version", "init", "doctor", "scorecard", "changelog", "explain",
	} {
		for _, flag := range localOnly {
			cmdName, flag := cmdName, flag
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
