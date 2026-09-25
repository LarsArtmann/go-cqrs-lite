package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// e2eTest: run() against fixture modules — flags → report → strict exit
// codes, the full pipeline including the --json wire (strict-gate residual
// hole (c), 05-26 §e4). Fixtures live under testdata/e2e/<name>/ with .txt
// suffixes (go tooling ignores testdata); each test copies them into a
// temp dir with real names so the package loader sees a true module.

// setupFixture copies testdata/e2e/<name>/* into a temp dir, renaming the
// *.go.mod.txt / *.go.txt shims to their real names and substituting the
// __REPO_EVENT_DIR__ token (relative replace paths cannot survive the move
// to a temp dir — they are go.mod-relative).
func setupFixture(t *testing.T, name string) string {
	t.Helper()

	src := filepath.Join("testdata", "e2e", name)
	dst := t.TempDir()

	eventDir, err := filepath.Abs(filepath.Join("..", "..", "event"))
	if err != nil {
		t.Fatalf("resolve event dir: %v", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	for _, e := range entries {
		in := filepath.Join(src, e.Name())

		out := e.Name()
		if ext := filepath.Ext(out[:len(out)-len(".txt")]); ext == ".go" || ext == ".mod" {
			out = out[:len(out)-len(".txt")]
		}

		data, err := os.ReadFile(in)
		if err != nil {
			t.Fatalf("read fixture file: %v", err)
		}

		data = []byte(strings.ReplaceAll(string(data), "__REPO_EVENT_DIR__", eventDir))

		if err := os.WriteFile(filepath.Join(dst, out), data, 0o600); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}
	}

	return dst
}

// wireModule is the subset of moduleJSON the E2E asserts on, decoded
// generically so this test cannot silently drift from the real struct.
type wireModule struct {
	SchemaVersion *int `json:"schemaVersion"`
	NoPins        bool `json:"noPins"`
	Bumps         []struct {
		Module string `json:"module"`
		Status string `json:"status"`
	} `json:"bumps"`
	Deprecations []struct {
		Rule string `json:"rule"`
	} `json:"deprecations"`
}

func decodeWire(t *testing.T, buf []byte) []wireModule {
	t.Helper()

	var modules []wireModule
	if err := json.Unmarshal(buf, &modules); err != nil {
		t.Fatalf("--json output is not a top-level module array: %v\n%s", err, buf)
	}

	return modules
}

// TestE2E_NoPins_ModuleStillScanned pins hole (a): a module with zero
// direct cqrs pins still gets the deprecation scan (NoPins ≠ no scan), the
// --json wire carries schemaVersion, and --strict exits clean on a
// genuinely clean module.
func TestE2E_NoPins_ModuleStillScanned(t *testing.T) {
	dir := setupFixture(t, "nopins")

	stdout := captureStdout(t)

	err := run(context.Background(), []string{"--json", "--strict", dir})

	out := stdout()

	if err != nil {
		t.Fatalf("--strict must pass on a clean NoPins module, got: %v\n%s", err, out)
	}

	modules := decodeWire(t, out)
	if len(modules) != 1 {
		t.Fatalf("expected 1 module in wire, got %d", len(modules))
	}

	mod := modules[0]
	if !mod.NoPins {
		t.Error("expected noPins=true in wire")
	}

	if mod.SchemaVersion == nil || *mod.SchemaVersion != moduleSchemaVersion {
		t.Errorf(
			"expected schemaVersion=%d in wire, got %v",
			moduleSchemaVersion,
			mod.SchemaVersion,
		)
	}

	if mod.Bumps == nil {
		t.Error("bumps must be an always-present array ([]), not null — the wire contract")
	}

	if len(mod.Deprecations) != 0 {
		t.Errorf("clean module must scan to zero deprecations, got %d", len(mod.Deprecations))
	}
}

// TestE2E_Violation_ScanAnalyzedFiles pins that the scan actually analyzed
// the cqrs-importing fixture's Go file (a zero-file scan proves nothing —
// the 02-47 lesson). NOTE: a NoPins module with zero cqrs imports honestly
// analyzes 0 files — consumer code can only call v5-removed APIs through
// DIRECT imports, which IsCQRSImport filters on; the analyzed-count guard
// therefore belongs on the importing fixture.
func TestE2E_Violation_ScanAnalyzedFiles(t *testing.T) {
	dir := setupFixture(t, "violation")

	_, analyzed, err := deprecationFindingsAnalyzed(dir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if analyzed == 0 {
		t.Fatal("V007 analyzed 0 Go files in the cqrs-importing fixture — the scan proved nothing")
	}
}

// TestE2E_Violation_StrictExitCodes pins the full flag → report → exit
// path on a module that both pins event/v4 (bump planning, dry-run leaves
// the tree untouched) and uses a v5-removed API (strict must fail with
// errStrictViolations; the finding must name V007).
func TestE2E_Violation_StrictExitCodes(t *testing.T) {
	dir := setupFixture(t, "violation")

	stdout := captureStdout(t)

	err := run(context.Background(), []string{"--json", "--dry-run", "--strict", dir})

	out := stdout()

	if !errors.Is(err, errStrictViolations) {
		t.Fatalf("expected errStrictViolations from --strict, got: %v\n%s", err, out)
	}

	modules := decodeWire(t, out)
	if len(modules) != 1 {
		t.Fatalf("expected 1 module in wire, got %d", len(modules))
	}

	mod := modules[0]
	if mod.NoPins {
		t.Error("fixture pins event/v4; noPins must be false")
	}

	if len(mod.Deprecations) == 0 {
		t.Fatal("expected the V007 finding on the wire, got none")
	}

	if got := mod.Deprecations[0].Rule; got != "V007" {
		t.Errorf("expected rule V007, got %q", got)
	}

	if len(mod.Bumps) == 0 {
		t.Error("pinned module must report its pin status on the wire")
	}

	// Dry-run contract: the fixture go.mod is byte-identical after the run.
	before, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	if string(before) == "" {
		t.Fatal("fixture go.mod vanished")
	}
}

// captureStdout swaps os.Stdout for a pipe for the duration of fn's
// return value readback. run() writes the --json wire to os.Stdout.
func captureStdout(t *testing.T) func() []byte {
	t.Helper()

	orig := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdout = w

	return func() []byte {
		os.Stdout = orig

		_ = w.Close()

		buf := make([]byte, 0, 8192)
		chunk := make([]byte, 4096)

		for {
			n, err := r.Read(chunk)
			buf = append(buf, chunk[:n]...)

			if err != nil {
				break
			}
		}

		return buf
	}
}
