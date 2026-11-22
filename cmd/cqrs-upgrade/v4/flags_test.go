package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"
)

// buildForTest wires the production CLI for in-process driving.
func buildForTest(t *testing.T) *cmdguard.CLI[AppConfig] {
	t.Helper()

	cli, err := buildCLI()
	if err != nil {
		t.Fatalf("buildCLI: %v", err)
	}

	return cli
}

// TestCLIParse_RejectsArgsAfterDir pins the 2026-09-11 hardening under the
// cmdguard parse layer: flags must precede the positional directory, so
// `cqrs-upgrade . --strict` fails loudly instead of silently ignoring
// --strict (stdlib flag's original sin) or silently applying it (pflag's
// interspersed default). SetInterspersed(false) on the root keeps the
// rejected tail visible as positional arguments.
func TestCLIParse_RejectsArgsAfterDir(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{".", "--strict"},
		{"--to", "v4.13.0", "/tmp", "--json"},
		{"/tmp", "extra-dir"},
	} {
		cli := buildForTest(t)

		err := cli.ExecuteWithArgs(context.Background(), args)
		if err == nil {
			t.Errorf("ExecuteWithArgs(%v) accepted args after the positional dir", args)
		} else if !errors.Is(err, errArgsAfterDir) {
			t.Errorf("ExecuteWithArgs(%v) = %v, want errArgsAfterDir", args, err)
		}
	}
}

// TestCLIParse_FlagsBeforeDirReachConfig proves the flags that PRECEDE the
// directory still land in the parsed config: the run fails on the second
// positional (so the pipeline never starts), then the config struct is
// inspected directly.
func TestCLIParse_FlagsBeforeDirReachConfig(t *testing.T) {
	t.Parallel()

	cli := buildForTest(t)

	err := cli.ExecuteWithArgs(context.Background(),
		[]string{"--strict", "--json", "/tmp", "second"})
	if !errors.Is(err, errArgsAfterDir) {
		t.Fatalf("expected errArgsAfterDir before any pipeline work, got: %v", err)
	}

	ac := cli.Config()
	if !ac.Strict || !ac.JSONOut {
		t.Errorf("flags before the dir must reach the config, got: %+v", *ac)
	}
}

// TestStrictGateError pins the gate ordering: an errored module fails FIRST
// (its pipeline died before any scan — unscanned is unproven), then a failed
// deprecation scan, then actual findings. A clean set passes.
func TestStrictGateError(t *testing.T) {
	t.Parallel()

	errored := moduleReport{Dir: "a", Error: "verify: build failed"}
	scanFailed := moduleReport{Dir: "b", ScanErr: errors.New("package load failed")}
	violating := moduleReport{Dir: "c", Deprecations: []findingJSON{{Position: "x.go:1:1"}}}

	err := strictGateError([]moduleReport{errored, scanFailed, violating})
	if err == nil || !errors.Is(err, errStrictModuleFailed) {
		t.Errorf("module error must fail the gate first, got %v", err)
	}

	err = strictGateError([]moduleReport{scanFailed, violating})
	if err == nil || !errors.Is(err, errStrictScanFailed) {
		t.Errorf("scan failure must fail the gate next, got %v", err)
	}

	err = strictGateError([]moduleReport{violating})
	if err == nil || !errors.Is(err, errStrictViolations) {
		t.Errorf("violations must fail the gate, got %v", err)
	}

	if err := strictGateError([]moduleReport{{Dir: "clean"}}); err != nil {
		t.Errorf("clean report set must pass, got %v", err)
	}

	msg := strictGateError([]moduleReport{errored}).Error()
	if !strings.Contains(msg, "unproven") {
		t.Errorf("module-error message should say readiness is unproven: %s", msg)
	}
}
