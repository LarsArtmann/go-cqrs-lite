package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestMultiModuleBuildContext_PartitionsProfiles is an integration test that
// runs the REAL loader (BuildContext → findGoModDirs → DetectFeaturesPerModule)
// against the go-cqrs-lite repo itself, which is a multi-module workspace with
// 79+ go.mod files. The unit tests use BuildContextFromSource (synthetic) which
// manually sets FeatureProfiles; this test verifies the actual loader wiring.
//
// It verifies:
//  1. BuildContext discovers multiple go.mod dirs
//  2. FeatureProfiles is populated with one entry per module
//  3. Each GoFile has a ModuleDir set (the loader attributes files to modules)
//  4. Different modules have different feature profiles (not all merged)
//  5. The deriver module (CommandFlow=commands) differs from library modules
func TestMultiModuleBuildContext_PartitionsProfiles(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join("..", "..")

	ctx, err := analyzer.BuildContext(repoRoot)
	if err != nil {
		t.Fatalf("BuildContext(%s): %v", repoRoot, err)
	}

	// 1. FeatureProfiles must be populated (multi-module workspace).
	if len(ctx.FeatureProfiles) < 10 {
		t.Fatalf(
			"expected FeatureProfiles to have 10+ entries for multi-module repo, got %d",
			len(ctx.FeatureProfiles),
		)
	}

	// 2. Verify the deriver module has CommandFlow=commands (it dispatches).
	deriverKey := filepath.Join(repoRoot, "deriver")
	deriverProfile, ok := ctx.FeatureProfiles[deriverKey]
	if !ok {
		t.Fatalf("deriver module not found in FeatureProfiles (key=%s)", deriverKey)
	}

	if deriverProfile.CommandFlow != analyzer.CommandFlowCommands {
		t.Errorf("deriver should have CommandFlow=commands, got %s", deriverProfile.CommandFlow)
	}

	// 3. Verify a pure library module (e.g. id/) has CommandFlow != commands.
	idKey := filepath.Join(repoRoot, "id")
	idProfile, ok := ctx.FeatureProfiles[idKey]
	if !ok {
		t.Fatalf("id module not found in FeatureProfiles (key=%s)", idKey)
	}

	if idProfile.CommandFlow == analyzer.CommandFlowCommands {
		t.Errorf("id module should NOT have CommandFlow=commands, got %s", idProfile.CommandFlow)
	}

	// 4. Verify these two modules have DIFFERENT profiles (proves per-module
	// partitioning works — under the old merged behavior, they'd be identical).
	if deriverProfile.CommandFlow == idProfile.CommandFlow {
		t.Errorf(
			"deriver and id have the same CommandFlow (%s) — profiles may be merged",
			deriverProfile.CommandFlow,
		)
	}

	// 5. Verify every GoFile has a ModuleDir set.
	missingModule := 0

	for _, gf := range ctx.GoFiles {
		if gf.ModuleDir == "" {
			missingModule++
		}
	}

	if missingModule > 0 {
		t.Errorf("%d GoFiles have empty ModuleDir — loader failed to attribute them", missingModule)
	}
}

// TestMultiModuleBuildContext_FileAttribution verifies that files from
// different modules are attributed to their correct ModuleDir.
func TestMultiModuleBuildContext_FileAttribution(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join("..", "..")

	ctx, err := analyzer.BuildContext(repoRoot)
	if err != nil {
		t.Fatalf("BuildContext(%s): %v", repoRoot, err)
	}

	// Find files from the deriver module.
	deriverDir := filepath.Join(repoRoot, "deriver")
	hasDeriverFiles := false

	for _, gf := range ctx.GoFiles {
		if gf.ModuleDir == deriverDir {
			hasDeriverFiles = true

			break
		}
	}

	if !hasDeriverFiles {
		// Check load errors — might be a transient resolution issue
		for _, e := range ctx.LoadErrors {
			if strings.Contains(e.Module, "deriver") {
				t.Skipf("deriver module failed to load (expected in some environments): %v", e.Errors)
			}
		}
		t.Errorf("no GoFiles attributed to %s — ModuleDir wiring is broken", deriverDir)
	}

	// Find files from the id module.
	idDir := filepath.Join(repoRoot, "id")
	hasIDFiles := false

	for _, gf := range ctx.GoFiles {
		if gf.ModuleDir == idDir {
			hasIDFiles = true

			break
		}
	}

	if !hasIDFiles {
		for _, e := range ctx.LoadErrors {
			if strings.Contains(e.Module, "/id") {
				t.Skipf("id module failed to load (expected in some environments): %v", e.Errors)
			}
		}
		t.Errorf("no GoFiles attributed to %s — ModuleDir wiring is broken", idDir)
	}
}
