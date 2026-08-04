package adttest

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// mockEngine implements only the base Engine interface — no backend interfaces.
// RunMatrix should skip all scenarios for this engine without panicking.
type mockEngine struct{}

func (mockEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: "mock"}
}

func (mockEngine) Close() error { return nil }

func TestRunMatrix_ZeroFactories(t *testing.T) {
	t.Parallel()

	// Should not panic, should not run any subtests.
	RunMatrix(t, nil)
}

func TestRunMatrix_SingleFactory(t *testing.T) {
	t.Parallel()

	// Single factory — no cross-engine parity check (len < 2).
	// Each scenario should run and pass for the memory engine.
	RunMatrix(t, []Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
	})
}

func TestRunMatrix_UnsupportedEngineSkips(t *testing.T) {
	t.Parallel()

	RunMatrix(t, []Factory{
		{Name: "mock", Create: func(t *testing.T) metaengine.Engine { return mockEngine{} }},
	})
	// All scenarios should be skipped — mockEngine implements no backends.
	// No panic, no failure.
}

func TestCanonicalizeAny_Nil(t *testing.T) {
	t.Parallel()

	if got := CanonicalizeAny(nil); got != "<nil>" {
		t.Errorf("CanonicalizeAny(nil) = %q, want %q", got, "<nil>")
	}
}

func TestCanonicalizeAny_String(t *testing.T) {
	t.Parallel()

	got := CanonicalizeAny("hello")
	want := `"hello"`
	if got != want {
		t.Errorf("CanonicalizeAny(%q) = %q, want %q", "hello", got, want)
	}
}

func TestCanonicalizeAny_MapSortedKeys(t *testing.T) {
	t.Parallel()

	// Keys must appear in sorted order regardless of insertion order.
	m := map[string]any{"zebra": 1, "apple": 2, "mango": 3}
	got := CanonicalizeAny(m)

	// apple must come before mango must come before zebra.
	applePos := indexOf(got, "apple")
	mangoPos := indexOf(got, "mango")
	zebraPos := indexOf(got, "zebra")

	if applePos >= mangoPos || mangoPos >= zebraPos {
		t.Errorf("CanonicalizeAny did not sort keys: apple=%d mango=%d zebra=%d in %q",
			applePos, mangoPos, zebraPos, got)
	}
}

func TestCanonicalizeAny_Slice(t *testing.T) {
	t.Parallel()

	s := []any{"a", "b", "c"}
	got := CanonicalizeAny(s)

	if got != `["a","b","c"]` {
		t.Errorf("CanonicalizeAny(slice) = %q, want %q", got, `["a","b","c"]`)
	}
}

func TestCanonicalizeCounter_NonCounterFallback(t *testing.T) {
	t.Parallel()

	// When passed a non-counter value, should delegate to CanonicalizeAny.
	got := CanonicalizeCounter("hello")
	want := CanonicalizeAny("hello")

	if got != want {
		t.Errorf("CanonicalizeCounter(non-counter) = %q, want %q", got, want)
	}
}

func TestCanonicalizeNeighbors_NonNeighborsFallback(t *testing.T) {
	t.Parallel()

	got := CanonicalizeNeighbors("hello")
	want := CanonicalizeAny("hello")

	if got != want {
		t.Errorf("CanonicalizeNeighbors(non-neighbors) = %q, want %q", got, want)
	}
}

func TestCanonicalizeScanResults_NonSliceFallback(t *testing.T) {
	t.Parallel()

	got := CanonicalizeScanResults("hello")
	want := CanonicalizeAny("hello")

	if got != want {
		t.Errorf("CanonicalizeScanResults(non-slice) = %q, want %q", got, want)
	}
}

func TestScenarios_AllElevenADTs(t *testing.T) {
	t.Parallel()

	scenarios := Scenarios()
	if len(scenarios) != 11 {
		t.Errorf("Scenarios() returned %d scenarios, want 11", len(scenarios))
	}

	seen := make(map[string]bool)
	for _, s := range scenarios {
		seen[s.Name] = true
		if s.Requires == "" {
			t.Errorf("Scenario %q has empty Requires", s.Name)
		}
		if s.Setup == nil || s.Read == nil || s.Canonicalize == nil {
			t.Errorf("Scenario %q has nil Setup/Read/Canonicalize", s.Name)
		}
	}

	for _, name := range []string{"Map", "Set", "Counter", "Graph", "SortedMap", "Log", "Multimap"} {
		if !seen[name] {
			t.Errorf("Scenarios() missing %q", name)
		}
	}
}

func TestBackendInterfaces_AllPresent(t *testing.T) {
	t.Parallel()

	required := []string{
		"MapBackend", "SetBackend", "CounterBackend",
		"GraphBackend", "ScanBackend", "LogBackend", "MultimapBackend",
	}

	for _, name := range required {
		if _, ok := backendInterfaces[name]; !ok {
			t.Errorf("backendInterfaces missing %q", name)
		}
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
