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
