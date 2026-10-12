package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected to a pipe and returns what
// was written. Not parallel-safe by design (process-global os.Stderr).
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	fn()

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("close writer: %v", closeErr)
	}
	os.Stderr = orig

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)

	return string(buf[:n])
}

// TestProfileForFile_FallbackWarnsOnce pins the warn-once hardening: a
// non-empty FeatureProfiles map where NOTHING matches the file path must
// downgrade to the primary profile LOUDLY (exactly one stderr warning), never
// silently — the silent shape hid the relative-key bug for a full wave
// (fixed 2026-10-12). A matching key must not warn.
func TestProfileForFile_FallbackWarnsOnce(t *testing.T) {
	primary := FeatureProfile{HasServer: true}
	mismatched := FeatureProfile{HasServer: false}

	ctx := &AnalysisContext{
		FeatureProfile:  primary,
		FeatureProfiles: map[string]FeatureProfile{"/repo/api": mismatched},
	}

	// A path no key matches: expect the warning exactly once across two calls.
	out := captureStderr(t, func() {
		got := ctx.ProfileForFile("/other/module/file.go")
		if got.HasServer != primary.HasServer {
			t.Errorf("fallback profile = %+v, want the primary", got)
		}
		got = ctx.ProfileForFile("/another/file.go")
		if got.HasServer != primary.HasServer {
			t.Errorf("second fallback profile = %+v, want the primary", got)
		}
	})

	warnings := strings.Count(out, "falling back to the primary module's profile")
	if warnings != 1 {
		t.Errorf("got %d fallback warnings, want exactly 1: %q", warnings, out)
	}

	// A path an existing key matches: no warning on this context.
	out = captureStderr(t, func() {
		got := ctx.ProfileForFile(filepath.Join("/repo/api/server.go"))
		if got.HasServer != mismatched.HasServer {
			t.Errorf("matched profile = %+v, want the per-module entry", got)
		}
	})
	if strings.Contains(out, "falling back to the primary") {
		t.Errorf("matched lookup must not warn: %q", out)
	}

	// A second context warns independently (per-analysis, not per-process).
	other := &AnalysisContext{
		FeatureProfile:  primary,
		FeatureProfiles: map[string]FeatureProfile{"/repo/api": mismatched},
	}
	out = captureStderr(t, func() {
		_ = other.ProfileForFile("/unrelated/x.go")
	})
	if !strings.Contains(out, "falling back to the primary") {
		t.Errorf("a fresh context must warn again: %q", out)
	}
}
