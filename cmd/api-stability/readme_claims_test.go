package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The README makes numeric claims about the library. These meta-tests keep
// those claims honest: when reality drifts (a dependency is added, coverage
// baselines move, a module appears), the tests fail until the README is
// updated — the same pattern as TestEveryGoModDirIsInModulesList, applied to
// marketing claims.

func readmePath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(".", "..", "..", "README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	return string(data)
}

// TestREADMEClaim_EventThirdPartyDeps verifies the "3 third-party
// dependencies" claim for the event module by resolving its actual production
// dependency graph (non-test, non-standard) and counting third-party modules
// (everything NOT under github.com/larsartmann).
func TestREADMEClaim_EventThirdPartyDeps(t *testing.T) {
	t.Parallel()

	eventDir := filepath.Join(".", "..", "..", "event")

	cmd := exec.Command("go", "list", "-deps",
		"-f", "{{if not .Standard}}{{.Module.Path}}{{end}}", ".")
	cmd.Dir = eventDir
	cmd.Env = append(os.Environ(), "GOWORK=off")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list in event module: %v", err)
	}

	thirdParty := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "github.com/larsartmann/go-cqrs-lite/event/v4" {
			continue
		}
		if strings.Contains(line, "github.com/larsartmann") {
			continue
		}

		thirdParty[line] = true
	}

	if len(thirdParty) != 3 {
		t.Errorf("event module resolves to %d third-party deps (%v), README claims 3 — "+
			"update README and this test together", len(thirdParty), thirdParty)
	}

	readme := readmePath(t)
	for _, claim := range []string{"only 3 third-party", "3 third-party deps"} {
		if !strings.Contains(readme, claim) {
			t.Errorf("README is missing the %q dependency claim", claim)
		}
	}
}

// TestREADMEClaim_ModuleCountFloor verifies the "80+ modules" claim against
// the actual go.mod count (vendor excluded).
func TestREADMEClaim_ModuleCountFloor(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")

	count := 0

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || (strings.HasPrefix(info.Name(), ".") && path != projectRoot) {
				return filepath.SkipDir
			}

			return nil
		}
		if info.Name() == "go.mod" {
			count++
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	if count < 80 {
		t.Errorf("repo has %d go.mod files, README claims 80+ — the claim overstates", count)
	}

	if !strings.Contains(readmePath(t), "80+ modules") && !strings.Contains(readmePath(t), "80+ independently-versioned modules") {
		t.Error("README no longer carries the 80+ module-count claim — update the meta-test")
	}
}

// TestREADMEClaim_CoverageFloors verifies every per-module coverage claim in
// the README against the check-coverage.sh baseline map: a README claim may
// UNDERSTATE the baseline (rounding down is honest) but never exceed it.
func TestREADMEClaim_CoverageFloors(t *testing.T) {
	t.Parallel()

	baselineData, err := os.ReadFile(filepath.Join(".", "..", "..", "scripts", "check-coverage.sh"))
	if err != nil {
		t.Fatalf("read check-coverage.sh: %v", err)
	}

	baselineLine := regexp.MustCompile(`\[([a-zA-Z/_-]+)\]=(\d+(?:\.\d+)?)`)
	baselines := map[string]float64{}
	for _, m := range baselineLine.FindAllStringSubmatch(string(baselineData), -1) {
		v, parseErr := strconv.ParseFloat(m[2], 64)
		if parseErr != nil {
			t.Fatalf("parse baseline %s: %v", m[2], parseErr)
		}

		baselines[m[1]] = v
	}

	claimLine := regexp.MustCompile(`\b(event|decider|dispatcher)\s+(\d+)%`)
	readme := readmePath(t)

	claimed := 0
	for _, m := range claimLine.FindAllStringSubmatch(readme, -1) {
		module := m[1]

		claim, parseErr := strconv.ParseFloat(m[2], 64)
		if parseErr != nil {
			t.Fatalf("parse claim %s: %v", m[2], parseErr)
		}

		base, ok := baselines[module]
		if !ok {
			t.Errorf("README claims coverage for %q but check-coverage.sh has no baseline for it", module)

			continue
		}
		if claim > base {
			t.Errorf("README claims %s at %.0f%% but the check-coverage floor is %.1f%% — "+
				"round claims DOWN to the floor", module, claim, base)
		}

		claimed++
	}

	if claimed < 3 {
		t.Errorf("found %d per-module coverage claims in README, expected the named set "+
			"(event, decider, dispatcher at minimum) — did the Maturity section change?", claimed)
	}

	if !strings.Contains(readme, "test coverage") {
		t.Error("README Maturity section no longer mentions test coverage")
	}

	fmt.Fprintf(os.Stderr, "coverage claims verified: %d (baselines: %v)\n", claimed, baselines)
}
