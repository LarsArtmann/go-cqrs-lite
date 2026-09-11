package main

import (
	"errors"
	"strings"
	"testing"
)

// TestParseFlags_RejectsArgsAfterDir pins the 2026-09-11 hardening: stdlib
// flag parsing stops at the first positional, so `cqrs-upgrade . --strict`
// used to silently ignore --strict and run as a plain report. Extra args
// after the directory must now fail loudly.
func TestParseFlags_RejectsArgsAfterDir(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{".", "--strict"},
		{"--to", "v4.13.0", "/tmp", "--json"},
		{"/tmp", "extra-dir"},
	} {
		if _, err := parseFlags(args); err == nil {
			t.Errorf("parseFlags(%v) accepted args after the positional dir", args)
		} else if !errors.Is(err, errArgsAfterDir) {
			t.Errorf("parseFlags(%v) = %v, want errArgsAfterDir", args, err)
		}
	}

	cfg, err := parseFlags([]string{"--strict", "--json", "/tmp"})
	if err != nil {
		t.Fatalf("flags before the dir must keep working: %v", err)
	}

	if !cfg.strict || !cfg.jsonOut || cfg.dir != "/tmp" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

// TestStrictGateError pins the gate ordering: a failed deprecation scan
// fails FIRST (unscannable = not proven clean), then actual findings. A
// clean set passes.
func TestStrictGateError(t *testing.T) {
	t.Parallel()

	scanFailed := moduleReport{Dir: "a", ScanErr: errors.New("package load failed")}
	violating := moduleReport{Dir: "b", Deprecations: []findingJSON{{Position: "x.go:1:1"}}}

	err := strictGateError([]moduleReport{scanFailed, violating})
	if err == nil || !errors.Is(err, errStrictScanFailed) {
		t.Errorf("scan failure must fail the gate first, got %v", err)
	}

	err = strictGateError([]moduleReport{violating})
	if err == nil || !errors.Is(err, errStrictViolations) {
		t.Errorf("violations must fail the gate, got %v", err)
	}

	if err := strictGateError([]moduleReport{{Dir: "clean"}}); err != nil {
		t.Errorf("clean report set must pass, got %v", err)
	}

	msg := strictGateError([]moduleReport{scanFailed}).Error()
	if !strings.Contains(msg, "unproven") {
		t.Errorf("scan-failure message should say readiness is unproven: %s", msg)
	}
}
