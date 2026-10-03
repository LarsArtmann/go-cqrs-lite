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
	if err := os.WriteFile(filepath.Join(dir, ".cqrs-lint.json"), []byte(configJSON), 0o644); err != nil {
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

	var out string
	captureStdout(t, func() {
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

	var out string
	captureStdout(t, func() {
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
// "Usage:" block of the root command's long help (`  cqrs-lint rules ...`).
func documentedCommands(t *testing.T, root *cobra.Command) map[string]string {
	t.Helper()

	documented := map[string]string{}
	re := regexp.MustCompile(`(?m)^  cqrs-lint ([a-z]+)\s+(.+)$`)

	for _, match := range re.FindAllStringSubmatch(root.Long, -1) {
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

	documented := documentedCommands(t, root)
	for _, name := range registeredCommands(root) {
		if _, ok := documented[name]; !ok {
			t.Errorf("subcommand %q is registered but missing from the root usage block", name)
		}
	}

	for name, desc := range documented {
		cmd, _, err := root.Find([]string{name})
		if err != nil || cmd == nil || cmd.Name() != name {
			t.Errorf("usage block documents %q but no such subcommand is registered", name)

			continue
		}

		if !strings.Contains(cmd.Short, desc) {
			t.Errorf(
				"usage block description for %q (%q) is not covered by the command's Short (%q) — they drifted",
				name, desc, cmd.Short,
			)
		}
	}
}

// TestCompletionBashSucceeds pins that cobra's auto-added completion command
// still executes under the local-flag scoping (an unknown-flag regression
// here would break shell setup for every user).
func TestCompletionBashSucceeds(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)

	var out string
	captureStdout(t, func() {
		if err := cli.ExecuteWithArgs(context.Background(), []string{"completion", "bash"}); err != nil {
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
	t.Parallel()

	cli := newTestCLI(t)

	var out string
	captureStdout(t, func() {
		if err := cli.ExecuteWithArgs(context.Background(), []string{"help", "doctor"}); err != nil {
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
		cmd      string
		formats  []string
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
				t.Errorf("%s vocabulary contains %q which the lint vocabulary rejects", tc.name, format)
			}
		}
	}
}
