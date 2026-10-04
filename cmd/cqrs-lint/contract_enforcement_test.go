package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"encoding/json/jsontext"

	"github.com/spf13/cobra"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// This file mechanically enforces the shipped CLI contract claims:
//
//  1. Config-file format parity: a .cqrs-lint.json setting "format": "json"
//     switches EVERY multi-format command to JSON, not just the lint run
//     (the README claims this; these tests make it true or fail).
//  2. Help-drift golden: every subcommand registered on the CLI appears in
//     the hand-written usage block of the root long help, and vice versa —
//     the `changelog` omission class can never recur silently.
//  3. Surface tests for cobra's auto-added `completion` and `help [cmd]`
//     under the local-flag scoping.
//  4. The changelog fallback reports honestly when no release tag exists.
//
// Flag vocabulary/acceptance matrices live in flag_contract_test.go.

// allSubcommands lists every user-registered subcommand (cobra's auto-added
// help/completion excluded — they get their own surface tests).
func allSubcommands() []string {
	return []string{"rules", "version", "init", "doctor", "scorecard", "changelog", "explain"}
}

// presetNamesForTest returns every named preset the init command accepts.
func presetNamesForTest(t *testing.T) []string {
	t.Helper()

	return analyzer.ValidPresetNames()
}

// formatFlagHelpTag returns the help text of the root --format flag.
func formatFlagHelpTag(t *testing.T) string {
	t.Helper()

	field, ok := reflect.TypeOf(AppConfig{}).FieldByName("Format")
	if !ok {
		t.Fatal("AppConfig.Format field not found")
	}

	return field.Tag.Get("help")
}

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
	if !jsontext.Value([]byte(trimmed)).IsValid() {
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
	if !jsontext.Value([]byte(trimmed)).IsValid() {
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
// here would break shell setup for every user). Not parallel: captureStdout
// swaps the process-wide os.Stdout.
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

// TestChangelogFallbackDistinguishesMissingTag pins the honest-fallback
// contract: with a version that has no release tag (dev/test builds), the
// changelog still emits the last 20 commits but REPORTS fallback=true so the
// CLI can tell the user — instead of silently truncating.
func TestChangelogFallbackDistinguishesMissingTag(t *testing.T) {
	t.Parallel()

	// "9.9.9" has no cmd/cqrs-lint/v9.9.9 tag in this repo; the test runs
	// inside the real git worktree so git log works.
	result, err := computeChangelog(context.Background(), "9.9.9")
	if err != nil {
		t.Fatalf("computeChangelog for missing tag: %v", err)
	}
	if !result.fallback {
		t.Error("missing tag must set fallback=true")
	}
	if len(result.commits) == 0 {
		t.Error("fallback must still emit the last 20 commits")
	}
}
