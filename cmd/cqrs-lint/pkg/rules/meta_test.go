package rules

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestAllDetectorsInstantiate verifies that every detector constructor
// returns a non-nil detector with a non-empty name. This is a compile-time
// and runtime guard against broken detector constructors (e.g., the b3931503
// incident where slices.Contains() was called with zero arguments).
func TestAllDetectorsInstantiate(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{}
	detectors := RegisterAll(ctx)

	if len(detectors) != 61 {
		t.Fatalf("expected 61 detectors, got %d", len(detectors))
	}

	for _, d := range detectors {
		if d == nil {
			t.Fatal("detector is nil")
		}

		name := d.Name()
		if name == "" {
			t.Fatal("detector has empty name")
		}

		// Every detector name must start with a rule ID prefix (e.g., "C001-")
		if len(name) < 5 {
			t.Fatalf("detector name too short: %q", name)
		}

		// Must start with uppercase letter + 3 digits + dash
		if name[0] < 'A' || name[0] > 'Z' {
			t.Fatalf("detector name must start with uppercase letter: %q", name)
		}

		for i := 1; i < 4; i++ {
			if name[i] < '0' || name[i] > '9' {
				t.Fatalf("detector name must have digits at positions 1-3: %q", name)
			}
		}
	}
}

// TestCriticalDetectorsInstantiate verifies the --fast mode subset.
func TestCriticalDetectorsInstantiate(t *testing.T) {
	t.Parallel()

	ctx := &analyzer.AnalysisContext{}
	detectors := RegisterCritical(ctx)

	if len(detectors) != 5 {
		t.Fatalf("expected 5 critical detectors, got %d", len(detectors))
	}

	for _, d := range detectors {
		if d == nil {
			t.Fatal("critical detector is nil")
		}

		if d.Name() == "" {
			t.Fatal("critical detector has empty name")
		}
	}
}
