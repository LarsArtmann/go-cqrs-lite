package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// doctorStaleModule writes a loadable one-package module that consumes
// go-cqrs-lite (via a replace to the repo, so no network is needed) and
// carries a stale //cqrs-lint:ignore directive — no logging library is
// imported, so D003 never fires there — and returns its directory.
func doctorStaleModule(t *testing.T) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	gomod := `module fixture

go 1.26

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0

replace github.com/larsartmann/go-cqrs-lite/event/v4 => ` + repoRoot + `/event/v4
`

	src := `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

//cqrs-lint:ignore(D003) stale: no logging library here
func nothing() {}

var _ = event.New
`

	dir := t.TempDir()

	for name, content := range map[string]string{"go.mod": gomod, "main.go": src} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("fixture go mod tidy: %v\n%s", err, out)
	}

	return dir
}

// TestDoctorJSON_FailOnStaleSuppressionsExitsNonZero pins the CI gate's
// exit-code contract on the JSON path: `doctor --format json
// --fail-on-stale-suppressions` must print the audit on stdout AND fail when
// a stale or unknown-rule directive exists. The JSON branch used to return
// before the flag was consulted, so the gate could never fail.
// Not parallel: captureStdout swaps the process-global os.Stdout.
func TestDoctorJSON_FailOnStaleSuppressionsExitsNonZero(t *testing.T) {
	dir := doctorStaleModule(t)
	cli := newTestCLI(t)

	var err error
	out := captureStdout(t, func() {
		// captureStdout already holds stdoutMu, which runCLI would re-lock.
		err = cli.ExecuteWithArgs(t.Context(), []string{
			"doctor", "--format", "json", "--fail-on-stale-suppressions", "--path", dir,
		})
	})

	if !errors.Is(err, errSuppressionAudit) {
		t.Fatalf("stale directive + --fail-on-stale-suppressions must fail the doctor, got: %v", err)
	}

	if !strings.Contains(out, `"stale": 1`) || !strings.Contains(out, `"audit"`) {
		t.Errorf("the JSON audit must still be printed before the failure, got:\n%s", out)
	}
}

// TestDoctorJSON_StaleWithoutFlagExitsZero is the control: the same project
// without --fail-on-stale-suppressions must still audit (implied) and succeed.
func TestDoctorJSON_StaleWithoutFlagExitsZero(t *testing.T) {
	dir := doctorStaleModule(t)
	cli := newTestCLI(t)

	var err error
	out := captureStdout(t, func() {
		err = cli.ExecuteWithArgs(t.Context(), []string{"doctor", "--format", "json", "--path", dir})
	})

	if err != nil {
		t.Fatalf("doctor --format json without the flag must succeed, got: %v", err)
	}

	if strings.Contains(out, `"audit"`) {
		t.Errorf("no audit flag must not attach an audit, got:\n%s", out)
	}
}
