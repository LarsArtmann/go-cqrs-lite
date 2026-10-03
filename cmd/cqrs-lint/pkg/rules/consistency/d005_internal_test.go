package consistency

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVersionParts_TrailingPunctuation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  []string
	}{
		{"v4.2.0.", []string{"4", "2", "0"}},
		{"v4.2.0,", []string{"4", "2", "0"}},
		{"v4.2.0", []string{"4", "2", "0"}},
		{"v4.0.x", []string{"4", "0", "x"}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got := parseVersionParts(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseVersionParts(%q) = %v, want %v", tc.input, got, tc.want)
			}

			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseVersionParts(%q)[%d] = %q, want %q",
						tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestIsVersionCompatible_TrailingDot(t *testing.T) {
	t.Parallel()

	if !isVersionCompatible("v4.2.0.", "v4.2.0") {
		t.Error("isVersionCompatible should treat trailing-dot version as compatible")
	}

	if !isVersionCompatible("v4.2.0", "v4.2.0") {
		t.Error("isVersionCompatible should match identical versions")
	}

	if isVersionCompatible("v4.2.0", "v4.3.0") {
		t.Error("isVersionCompatible should reject different versions")
	}
}

func TestReadGoModCQRSVersion_PrefersDirectOverIndirect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goMod := `module example.com/app

go 1.26

require (
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/dispatcher/v4 v4.2.0 // indirect
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0 // indirect
)
`
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	got := readGoModCQRSVersion(path)
	if got != "v4.2.0" {
		t.Fatalf("readGoModCQRSVersion with mixed direct/indirect = %q, want %q", got, "v4.2.0")
	}
}

func TestReadGoModCQRSVersion_FallsBackToIndirect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goMod := `module example.com/app

go 1.26

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.1.0 // indirect
)
`
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	got := readGoModCQRSVersion(path)
	if got != "v4.1.0" {
		t.Fatalf("readGoModCQRSVersion indirect-only = %q, want %q", got, "v4.1.0")
	}
}

func TestExtractCQRSVersion_SkipsCodeBlocks(t *testing.T) {
	t.Parallel()

	content := "This project uses go-cqrs-lite.\n" +
		"```go\n" +
		"import \"go-cqrs-lite/command/v4.2.0\"\n" +
		"```\n"
	got := extractCQRSVersion(content, "v4.3.0", nil)
	if got != "v4.3.0" {
		t.Fatalf(
			"extractCQRSVersion with code block = %q, want %q (code blocks must be skipped)",
			got,
			"v4.3.0",
		)
	}
}

func TestExtractCQRSVersion_SkipsImportPaths(t *testing.T) {
	t.Parallel()

	content := "go-cqrs-lite/command/v4.2.0 provides commands."
	got := extractCQRSVersion(content, "v4.3.0", nil)
	if got != "v4.3.0" {
		t.Fatalf(
			"extractCQRSVersion with import path = %q, want %q (version preceded by / must be skipped)",
			got,
			"v4.3.0",
		)
	}
}

func TestExtractCQRSVersion_SkipsPseudoVersions(t *testing.T) {
	t.Parallel()

	content := "go-cqrs-lite v4.2.1-0.20260808200723-546259830b28 is used."
	got := extractCQRSVersion(content, "v4.3.0", nil)
	if got != "v4.3.0" {
		t.Fatalf(
			"extractCQRSVersion with pseudo-version = %q, want %q (pseudo-versions must be skipped)",
			got,
			"v4.3.0",
		)
	}
}

func TestReadGoModCQRSVersion_OldBugReturnedIndirect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Regression: the old code took parts[len(parts)-1] which was "indirect"
	// when the line had a trailing comment.
	goMod := `module example.com/app

go 1.26

require (
	github.com/larsartmann/go-cqrs-lite/command/v4 v4.2.0 // indirect
)
`
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	got := readGoModCQRSVersion(path)
	if got == "indirect" {
		t.Fatal("readGoModCQRSVersion returned 'indirect' — the old bug is back")
	}
	if got != "v4.2.0" {
		t.Fatalf("readGoModCQRSVersion = %q, want %q", got, "v4.2.0")
	}
}

// --- #43: positional attachment ---

// TestExtractCQRSVersion_OtherModuleVersionFirst pins the #43 false positive
// where a sibling module's version on the same line was picked as the
// go-cqrs-lite claim: the FIRST version token belonged to go-finding, the
// true claim sat after the go-cqrs-lite mention.
func TestExtractCQRSVersion_OtherModuleVersionFirst(t *testing.T) {
	t.Parallel()

	content := "# App\n\nThis project pins go-finding v1.12.0 and go-cqrs-lite v4.12.1 for storage.\n"
	got := extractCQRSVersion(content, "v4.12.1", nil)
	if got != "v4.12.1" {
		t.Fatalf(
			"extractCQRSVersion other-module-first = %q, want %q (attachment must skip go-finding's token)",
			got,
			"v4.12.1",
		)
	}
}

// TestExtractCQRSVersion_HistoricalMention pins the #43 false positive where
// a past-state sentence ("upgraded from go-cqrs-lite v4.11.1") was reported
// as the doc's current-version claim.
func TestExtractCQRSVersion_HistoricalMention(t *testing.T) {
	t.Parallel()

	content := "# App\n\nUpgraded from go-cqrs-lite v4.11.1 to pick up storage fixes.\n"
	got := extractCQRSVersion(content, "v4.12.1", nil)
	if got != "v4.12.1" {
		t.Fatalf(
			"extractCQRSVersion historical = %q, want %q ('from' cue must drop the claim)",
			got,
			"v4.12.1",
		)
	}
}

// TestExtractCQRSVersion_DirectAttachmentKept proves the rule still reads the
// plain current-claim form — the stale-version detection depends on it.
func TestExtractCQRSVersion_DirectAttachmentKept(t *testing.T) {
	t.Parallel()

	content := "# App\n\nUses go-cqrs-lite v3.1.0\n"
	got := extractCQRSVersion(content, "v4.2.0", nil)
	if got != "v3.1.0" {
		t.Fatalf(
			"extractCQRSVersion direct = %q, want %q (adjacent claim must still count)",
			got,
			"v3.1.0",
		)
	}
}

// --- per-module pins (go-cqrs-lite tags are per-module; CV feedback
// 2026-10-03: accurate per-module doc references misread as stale) ---

// TestExtractCQRSVersion_PerModuleReferencesFresh: a doc listing each
// module's own pinned version ("pinned: system v4.7.0, decider v4.6.0,
// metaengine v4.13.0") is fully accurate and must NOT report a stale
// version, even though the versions differ from each other.
func TestExtractCQRSVersion_PerModuleReferencesFresh(t *testing.T) {
	t.Parallel()

	pins := map[string]string{
		"system":     "v4.7.0",
		"decider":    "v4.6.0",
		"metaengine": "v4.13.0",
	}

	content := "# App\n\nUses go-cqrs-lite (pinned: system v4.7.0, " +
		"decider v4.6.0, metaengine v4.13.0 — the Phase-0-proven set).\n"
	got := extractCQRSVersion(content, "v4.7.0", pins)
	if got != "v4.7.0" {
		t.Fatalf(
			"extractCQRSVersion per-module fresh = %q, want %q (all references match their pins)",
			got,
			"v4.7.0",
		)
	}
}

// TestExtractCQRSVersion_PerModuleStaleReference: a doc claiming a version
// for a NAMED module that differs from that module's pin stays a finding.
func TestExtractCQRSVersion_PerModuleStaleReference(t *testing.T) {
	t.Parallel()

	pins := map[string]string{
		"system":  "v4.7.0",
		"decider": "v4.6.0",
	}

	content := "# App\n\nUses go-cqrs-lite system v4.5.0 and decider v4.6.0.\n"
	got := extractCQRSVersion(content, "v4.7.0", pins)
	if got != "v4.5.0" {
		t.Fatalf(
			"extractCQRSVersion per-module stale = %q, want %q (system v4.5.0 vs pin v4.7.0)",
			got,
			"v4.5.0",
		)
	}
}

// TestExtractCQRSVersion_AnyPinMatchFresh: a version token with no adjacent
// module name is fresh when it matches ANY pinned module version — several
// different current versions are the ecosystem's normal shape.
func TestExtractCQRSVersion_AnyPinMatchFresh(t *testing.T) {
	t.Parallel()

	pins := map[string]string{
		"event":   "v4.12.0",
		"decider": "v4.7.0",
	}

	content := "# App\n\ngo-cqrs-lite v4.7.0 powers the decider fold.\n"
	got := extractCQRSVersion(content, "v4.12.0", pins)
	if got != "v4.12.0" {
		t.Fatalf(
			"extractCQRSVersion any-pin = %q, want %q (v4.7.0 matches the decider pin)",
			got,
			"v4.12.0",
		)
	}
}

// TestReadGoModCQRSVersionSet_PrefersDirectOverIndirect pins the per-module
// direct-over-indirect preference and the shortname derivation.
func TestReadGoModCQRSVersionSet_PrefersDirectOverIndirect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	goMod := `module example.com/app

go 1.26

require (
	github.com/larsartmann/go-cqrs-lite/system/v4 v4.10.0
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.11.0 // indirect
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.9.0 // indirect
)
`
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	got := readGoModCQRSVersionSet(path)
	want := map[string]string{
		"system":  "v4.10.0",
		"event":   "v4.11.0",
		"decider": "v4.9.0",
	}

	if len(got) != len(want) {
		t.Fatalf("readGoModCQRSVersionSet = %v, want %v", got, want)
	}

	for k, v := range want {
		if got[k] != v {
			t.Fatalf("readGoModCQRSVersionSet[%q] = %q, want %q", k, got[k], v)
		}
	}
}
