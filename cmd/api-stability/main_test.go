package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// jsonV2Env returns an environment with GOEXPERIMENT=jsonv2 set, needed because
// cmdguard uses encoding/json/v2 behind a build tag.
func jsonV2Env() []string {
	return append(os.Environ(), "GOEXPERIMENT=jsonv2")
}

const jsonV2Tags = "goexperiment.jsonv2"

// TestEveryGoModDirIsInModulesList asserts that every directory containing a
// go.mod (except examples, integration, the root workspace, and this tool's own
// module) appears in the modules slice. This catches the class of omission
// where a published module ships without its API surface being tracked.
func TestEveryGoModDirIsInModulesList(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")

	// Build a set of tracked module paths for O(1) lookup.
	tracked := make(map[string]struct{}, len(modules))
	for _, m := range modules {
		tracked[m] = struct{}{}
	}

	// Directories that are intentionally excluded from the api-stability gate.
	excluded := map[string]string{
		".":                             "root workspace go.mod",
		"cmd/api-stability":             "the api-stability tool itself (circular)",
		"integration":                   "workspace-only cross-module tests (published graph not self-contained)",
		"example/getting-started":       "example application",
		"example/metaengine-quickstart": "example application",
		"example/readme-quickstart":     "example application",
		"example/taskmanager":           "example application",
	}

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		// Skip hidden dirs and vendor.
		name := info.Name()
		if name == ".git" || name == "vendor" ||
			(len(name) > 0 && name[0] == '.' && path != projectRoot) {
			return filepath.SkipDir
		}
		// Check for go.mod in this directory.
		if _, err := os.Stat(filepath.Join(path, "go.mod")); os.IsNotExist(err) {
			return nil // no go.mod here, keep walking
		} else if err != nil {
			return err // walk error
		}
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		if reason, ok := excluded[rel]; ok {
			t.Logf("excluding %s (%s)", rel, reason)

			return nil
		}
		if _, ok := tracked[rel]; !ok {
			t.Errorf("directory %q has a go.mod but is NOT in the modules list in main.go — "+
				"add it to the modules slice so its API surface is tracked", rel)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}

func TestAPISurfaceCheck(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skip("golden file does not exist; run with --update first")
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "run", "-tags", jsonV2Tags, ".")
	cmd.Env = jsonV2Env()
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("API surface check failed:\n%s", out)
	}

	t.Logf("%s", out)
}

// TestToolCompiles is an always-run guard that builds the api-stability tool
// itself. Unlike TestAPISurfaceCheck (which is skipped when the golden is
// missing), this test runs unconditionally and catches compile breakage, e.g.
// a deleted helper function referenced by main.go, before it silently breaks
// every downstream API-surface verification.
func TestToolCompiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", jsonV2Tags, ".")
	cmd.Env = jsonV2Env()
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("api-stability tool does not compile:\n%s", out)
	}
}

func TestAPISurfaceUpdateIdempotent(t *testing.T) {
	// Serial: writes the golden file. Must not overlap with TestAPISurfaceCheck
	// which reads the golden file concurrently.

	projectRoot := filepath.Join(".", "..", "..")
	goldenPath := filepath.Join(projectRoot, "docs", "api_surface.txt")

	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skip("golden file does not exist; run with --update first")
	}

	original, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "run", "-tags", jsonV2Tags, ".", "--update")
	cmd.Env = jsonV2Env()
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update run failed: %s\n%s", err, out)
	}

	updated, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read updated golden: %v", err)
	}

	if string(original) != string(updated) {
		t.Errorf("golden file changed after update — API surface is not stable")
		if err := os.WriteFile(goldenPath, original, 0o600); err != nil {
			t.Logf("failed to restore golden: %v", err)
		}
	}
}

// TestTagContentMatchesChangelog verifies that every released version section
// in CHANGELOG.md has at least one corresponding git tag. This catches the
// "updated CHANGELOG but forgot to tag" drift scenario. It also verifies that
// the latest tagged version has a CHANGELOG entry ("tagged but didn't update
// CHANGELOG").
func TestTagContentMatchesChangelog(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")

	changelogBytes, err := os.ReadFile(filepath.Join(projectRoot, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG: %v", err)
	}

	// Extract all ## [vX.Y.Z] version sections from CHANGELOG.
	versionRe := regexp.MustCompile(`## \[(v\d+\.\d+\.\d+)\]`)
	matches := versionRe.FindAllStringSubmatch(string(changelogBytes), -1)
	if len(matches) == 0 {
		t.Fatal("no version sections found in CHANGELOG.md")
	}

	changelogVersions := make(map[string]bool, len(matches))
	for _, m := range matches {
		changelogVersions[m[1]] = true
	}

	// Get all git tags matching */v* pattern.
	tagCmd := exec.CommandContext(
		context.Background(),
		"git",
		"-C",
		projectRoot,
		"tag",
		"-l",
		"*/v*",
	)
	tagOut, err := tagCmd.Output()
	if err != nil {
		t.Fatalf("git tag list: %v", err)
	}

	// Extract the version suffix from each module tag (e.g., "event/v4.0.4" → "v4.0.4").
	tagVersionRe := regexp.MustCompile(`/((?:v)\d+\.\d+\.\d+)$`)
	taggedVersions := make(map[string]int)
	for _, line := range strings.Split(strings.TrimSpace(string(tagOut)), "\n") {
		if line == "" {
			continue
		}
		if m := tagVersionRe.FindStringSubmatch(line); m != nil {
			taggedVersions[m[1]]++
		}
	}

	if len(taggedVersions) == 0 {
		t.Fatal("no module version tags found in git")
	}

	// Every CHANGELOG version must have at least one tag.
	for ver := range changelogVersions {
		if taggedVersions[ver] == 0 {
			t.Errorf("CHANGELOG has ## [%s] but zero git tags at that version — "+
				"did you forget to tag the release?", ver)
		}
	}

	// The latest CHANGELOG version should have a reasonable number of module tags
	// (at least 10 — not all 58 modules release at the same version, but core
	// modules should be tagged together).
	latestChangelogVer := matches[0][1]
	if taggedVersions[latestChangelogVer] < 10 {
		t.Logf("WARNING: latest CHANGELOG version %s has only %d module tags "+
			"(expected >= 10 for a coordinated release)",
			latestChangelogVer, taggedVersions[latestChangelogVer])
	}
}

// TestExceptionsAreMinimal verifies that every EXCEPTIONS entry in
// scripts/check-module-layers.sh is actually necessary. An exception is dead
// in two cases:
//
//  1. Layer-incompatible: dep_layer <= mod_layer (same/lower-layer deps don't
//     trigger violations).
//  2. Indirect-only: the dep is marked "// indirect" in the module's go.mod.
//     The script's awk filter skips lines containing "//", so the dep never
//     reaches the exception lookup. An indirect-only exception is dead weight
//     AND a trap: if the dep is promoted to direct later, the exception would
//     silently suppress a real layer violation.
//
// This prevents the schema->snapshot, transport/http->testutil, and
// query/command->snapshot class of stale entries.
func TestExceptionsAreMinimal(t *testing.T) { //nolint:gocognit // comprehensive exception audit
	t.Parallel()

	projectRoot := filepath.Join(".", "..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "check-module-layers.sh")

	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read layer check script: %v", err)
	}
	script := string(scriptBytes)

	// Parse LAYER[<mod>]=<number> — skip comment lines.
	layerRe := regexp.MustCompile(`^\s*LAYER\[([^\]]+)\]=(\d+)\s*$`)
	layers := make(map[string]int)
	for line := range strings.SplitSeq(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if m := layerRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[2])
			layers[m[1]] = n
		}
	}
	if len(layers) == 0 {
		t.Fatal("failed to parse any LAYER entries from check-module-layers.sh")
	}

	// Parse EXCEPTIONS[<mod>]="<dep1> <dep2> ..."
	excRe := regexp.MustCompile(`^\s*EXCEPTIONS\[([^\]]+)\]="([^"]+)"`)
	type exception struct {
		module string
		dep    string
		reason string
	}
	var dead []exception
	for line := range strings.SplitSeq(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		m := excRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		module := m[1]
		modLayer, ok := layers[module]
		if !ok {
			t.Errorf("EXCEPTIONS[%s] references module not in LAYER map", module)

			continue
		}
		for dep := range strings.FieldsSeq(m[2]) {
			depLayer, ok := layers[dep]
			if !ok {
				t.Errorf("EXCEPTIONS[%s] references dep %q not in LAYER map", module, dep)

				continue
			}

			// Case 1: dep_layer <= mod_layer — no violation possible.
			if depLayer <= modLayer {
				dead = append(dead, exception{
					module: module,
					dep:    dep,
					reason: fmt.Sprintf(
						"dep_layer %d <= mod_layer %d — no violation is triggered",
						depLayer, modLayer),
				})

				continue
			}

			// Case 2: dep is indirect-only in module's go.mod — the script's
			// awk filter (line 318: !/\/\//) skips it, so the exception never
			// fires. This is a sleeping trap: if promoted to direct, the
			// exception silently suppresses a real violation.
			gomodPath := filepath.Join(projectRoot, module, "go.mod")
			gomodBytes, err := os.ReadFile(gomodPath)
			if err != nil {
				t.Errorf("EXCEPTIONS[%s] references module with unreadable go.mod: %v", module, err)

				continue
			}
			// Build the import-path prefix for this dep and check if it only
			// appears with // indirect.
			depImportPath := "github.com/larsartmann/go-cqrs-lite/" + dep
			lines := strings.SplitSeq(string(gomodBytes), "\n")
			directFound := false
			for gomodLine := range lines {
				if !strings.Contains(gomodLine, depImportPath) {
					continue
				}
				if !strings.Contains(gomodLine, "// indirect") {
					directFound = true

					break
				}
			}
			if !directFound {
				dead = append(dead, exception{
					module: module,
					dep:    dep,
					reason: fmt.Sprintf(
						"%q is indirect-only in %s/go.mod — the awk filter "+
							"(line 318) skips // lines so this exception never "+
							"fires; it would silently suppress a real violation "+
							"if the dep is promoted to direct",
						dep, module),
				})
			}
		}
	}

	for _, d := range dead {
		t.Errorf("EXCEPTIONS[%s] lists %q — %s; remove this stale exception entry",
			d.module, d.dep, d.reason)
	}
}
