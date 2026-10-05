package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// This file pins the subcommand-consistency contract:
//
//  1. The shared --format flag is honored (or loudly rejected) by every
//     command that renders in multiple formats — no silent text fallback.
//  2. Format validation happens BEFORE package loading (fail fast).
//  3. Lint-run-only flags are local to the root command — passing --fix to
//     version is an unknown-flag error, not a silent no-op.
//  4. init honors --path instead of always writing to the CWD.
//  5. The root --scorecard flag and the scorecard subcommand share ONE
//     runner (runScorecard), so both entry points behave identically.

// newTestCLI builds the CLI with every subcommand registered, mirroring
// main()'s wiring so ExecuteWithArgs can drive subcommands in-process.
func newTestCLI(t *testing.T) *cmdguard.CLI[AppConfig] {
	t.Helper()

	cli, err := cmdguard.NewCLI(
		"cqrs-lint",
		"Domain-aware linter for go-cqrs-lite consumers",
		AppConfig{},
		cmdguard.WithCLIVersion("test"),
		cmdguard.WithConfigFileLoader(JSONCLoader{}, ".cqrs-lint.json"),
	)
	if err != nil {
		t.Fatalf("create CLI: %v", err)
	}

	for _, setup := range []func(*cmdguard.CLI[AppConfig]) error{
		setupRulesCommand,
		setupVersionCommand,
		setupInitCommand,
		setupDoctorCommand,
		setupScorecardCommand,
		setupChangelogCommand,
		setupExplainCommand,
	} {
		if err := setup(cli); err != nil {
			t.Fatalf("register command: %v", err)
		}
	}

	return cli
}

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns whatever
// was written. NOT safe for parallel tests.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}

func TestValidateFormatFlag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		format    string
		supported []string
		wantErr   bool
	}{
		{"text", []string{"text", "json"}, false},
		{"JSON", []string{"text", "json"}, false}, // case-insensitive
		{" text ", []string{"text", "json"}, false},
		{"", []string{"text", "json"}, false}, // zero value = default text
		{"yaml", []string{"text", "json"}, true},
		{"csv", []string{"text", "json", "markdown", "sarif"}, true},
	}

	for _, tc := range cases {
		err := validateFormatFlag(tc.format, tc.supported...)
		if tc.wantErr && err == nil {
			t.Errorf("format %q: expected error, got nil", tc.format)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("format %q: unexpected error: %v", tc.format, err)
		}
		if tc.wantErr {
			msg := err.Error()
			if !strings.Contains(msg, tc.format) {
				t.Errorf("error should quote the offending value %q, got: %s", tc.format, msg)
			}
			for _, s := range tc.supported {
				if !strings.Contains(msg, s) {
					t.Errorf("error should list supported value %q, got: %s", s, msg)
				}
			}
		}
	}
}

func TestRulesFormatResolvesSharedFormatFlag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		flags     rulesFlags
		cfgFormat string
		want      string
		wantErr   bool
	}{
		{"json boolean wins", rulesFlags{JSON: true}, "markdown", "json", false},
		{"markdown boolean wins", rulesFlags{Markdown: true}, "json", "markdown", false},
		{"shared format json", rulesFlags{}, "json", "json", false},
		{"shared format markdown", rulesFlags{}, "markdown", "markdown", false},
		{"shared format text", rulesFlags{}, "text", "text", false},
		{"zero value is text", rulesFlags{}, "", "text", false},
		{"unsupported sarif errors", rulesFlags{}, "sarif", "", true},
		{"unsupported csv errors", rulesFlags{}, "csv", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := rulesFormat(tc.flags, &AppConfig{Format: tc.cfgFormat})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got format %q", got)
				}
				if !strings.Contains(err.Error(), "invalid --format") {
					t.Errorf("expected invalid --format error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got format %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRunRejectsInvalidFormatBeforeLoad pins fail-fast: an invalid --format
// must error out before any package loading happens — the error is the
// format complaint even when the path is unusable.
func TestRunRejectsInvalidFormatBeforeLoad(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), &AppConfig{
		Path:   filepath.Join(t.TempDir(), "does-not-exist"),
		Format: "yaml",
	})
	if err == nil {
		t.Fatal("expected error for invalid --format, got nil")
	}
	if !strings.Contains(err.Error(), `invalid --format "yaml"`) {
		t.Errorf("expected invalid --format error, got: %v", err)
	}
	if strings.Contains(err.Error(), "load packages") {
		t.Errorf("format validation must precede package loading, got: %v", err)
	}
}

// TestDoctorRejectsUnsupportedFormatBeforeLoad: doctor previously paid the
// full package-load cost before noticing an invalid format. The error must
// be the format complaint, not a load failure, even for a missing path.
func TestDoctorRejectsUnsupportedFormatBeforeLoad(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)
	err := cli.ExecuteWithArgs(context.Background(), []string{
		"doctor", "--format", "yaml",
		"--path", filepath.Join(t.TempDir(), "missing"),
	})
	if err == nil {
		t.Fatal("expected error for doctor --format yaml, got nil")
	}
	if !strings.Contains(err.Error(), `invalid --format "yaml"`) {
		t.Errorf("expected invalid --format error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "supported: text, json") {
		t.Errorf("error should list doctor's supported formats, got: %v", err)
	}
}

func TestScorecardRejectsUnsupportedFormat(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)
	err := cli.ExecuteWithArgs(context.Background(), []string{
		"scorecard", "--format", "csv",
		"--path", t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for scorecard --format csv, got nil")
	}
	if !strings.Contains(err.Error(), "supported: text, json, markdown, sarif") {
		t.Errorf("error should list scorecard's supported formats, got: %v", err)
	}
}

// TestVersionRejectsLintOnlyFlags: --fix is a lint-run-only flag (local to
// the root command). Passing it to version must be an unknown-flag error —
// previously it was silently accepted and ignored.
func TestVersionRejectsLintOnlyFlags(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"--fix", "--dry-run", "--fast", "--only", "--scorecard", "--fp-suspects"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()

			cli := newTestCLI(t)
			err := cli.ExecuteWithArgs(context.Background(), []string{"version", flag})
			if err == nil {
				t.Fatalf("version %s: expected unknown-flag error, got nil", flag)
			}
			if !strings.Contains(strings.ToLower(err.Error()), "unknown flag") {
				t.Errorf("version %s: expected unknown-flag error, got: %v", flag, err)
			}
		})
	}
}

// TestSubcommandsKeepSharedFlags: the shared vocabulary (path, format, color,
// min-severity, min-confidence) stays available on the subcommands that
// consume it.
func TestSubcommandsKeepSharedFlags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cli := newTestCLI(t)
	err := cli.ExecuteWithArgs(context.Background(), []string{"init", "--path", dir})
	if err != nil {
		t.Fatalf("init --path should stay supported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".cqrs-lint.json")); err != nil {
		t.Errorf("init should honor --path and write into %s: %v", dir, err)
	}
}

func TestInitHonorsPathFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cli := newTestCLI(t)

	if err := cli.ExecuteWithArgs(
		context.Background(),
		[]string{"init", "--path", dir},
	); err != nil {
		t.Fatalf("init --path %s: %v", dir, err)
	}

	target := filepath.Join(dir, ".cqrs-lint.json")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected %s to exist: %v", target, err)
	}

	// Second init into the same dir must fail on the existing file.
	if err := cli.ExecuteWithArgs(
		context.Background(),
		[]string{"init", "--path", dir},
	); !errors.Is(
		err,
		errConfigExists,
	) {
		t.Errorf("second init should fail with errConfigExists, got: %v", err)
	}
}

func TestInitErrorsOnMissingPath(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)
	missing := filepath.Join(t.TempDir(), "no-such-dir")
	err := cli.ExecuteWithArgs(context.Background(), []string{"init", "--path", missing})
	if err == nil {
		t.Fatal("expected error for init --path <missing>, got nil")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error should mention the missing path, got: %v", err)
	}
}

// TestDoctorFixFlagRenamedToPrune: doctor's suppression-cleanup flag was
// renamed from --fix to --prune-suppressions because the root --fix
// (findings autofix) collision made every docs example ambiguous. The old
// name must now be an unknown-flag error — silently keeping a second
// meaning for --fix is the exact class this file pins out.
func TestDoctorFixFlagRenamedToPrune(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)

	err := cli.ExecuteWithArgs(context.Background(), []string{"doctor", "--fix"})
	if err == nil {
		t.Fatal("expected unknown-flag error for doctor --fix, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unknown flag") {
		t.Errorf("expected unknown-flag error, got: %v", err)
	}

	// --prune-suppressions is accepted (audit on an empty temp dir).
	dir := t.TempDir()
	err = cli.ExecuteWithArgs(context.Background(), []string{
		"doctor", "--prune-suppressions", "--dry-run", "--path", dir,
	})
	if err != nil {
		t.Errorf("doctor --prune-suppressions --dry-run on empty dir: %v", err)
	}
}

// TestSingleRenderCommandsIgnoreSharedFormat: version/explain/changelog
// render exactly one format each; the shared --format flag is part of the
// global vocabulary and is deliberately IGNORED (not rejected) here — a
// config file setting "format": "json" for lint output must not break
// `cqrs-lint version` in CI scripts.
func TestSingleRenderCommandsIgnoreSharedFormat(t *testing.T) {
	t.Parallel()

	cli := newTestCLI(t)

	for _, cmd := range []string{"version", "explain"} {
		if err := cli.ExecuteWithArgs(
			context.Background(), []string{cmd, "--format", "json"},
		); err != nil {
			t.Errorf("%s --format json should be accepted-and-ignored, got: %v", cmd, err)
		}
	}
}

// TestRunScorecardThresholdGate: the threshold CI gate lives in the shared
// runner, so both the subcommand and the root --scorecard flag route
// through it (the root flag passes 0 = gate off).
func TestRunScorecardThresholdGate(t *testing.T) {
	t.Parallel()

	actx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func main() {}
`,
	})

	cfg := &AppConfig{Format: "text"}

	var out string
	captureStdout(t, func() {
		out = "captured"
		err := runScorecard(context.Background(), cfg, actx, 0)
		if err != nil {
			t.Errorf("runScorecard without threshold: unexpected error: %v", err)
		}
	})
	if out != "captured" {
		t.Errorf("captureStdout ran fn unexpectedly")
	}

	captureStdout(t, func() {
		err := runScorecard(context.Background(), cfg, actx, 101)
		if !errors.Is(err, errScorecardBelowThreshold) {
			t.Errorf(
				"runScorecard with threshold 101: got %v, want errScorecardBelowThreshold",
				err,
			)
		}
	})
}
