package metaengine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestUniversalADT_DegradedDiagnosticShowsCostPenalty verifies the degraded
// diagnostic message includes an estimated latency penalty.
func TestUniversalADT_DegradedDiagnosticShowsCostPenalty(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:      "degraded-cost-test",
		NsPerRead: 500,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTMap: true,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if d.Level != metaengine.DiagLevelDegraded {
			continue
		}

		if !strings.Contains(d.Message, "est ") {
			t.Errorf("degraded diagnostic should include cost estimate, got: %s", d.Message)
		}

		if !strings.Contains(d.Message, "ms") {
			t.Errorf("degraded diagnostic should include latency in ms, got: %s", d.Message)
		}

		return
	}

	t.Error("expected a DEGRADED diagnostic with cost penalty")
}

// TestUniversalADT_DegradedRecommendsNativeEngine verifies that when a degraded
// engine is selected (due to lower total cost) but a native engine for the same
// ADT is available in the store, the diagnostic recommends it.
func TestUniversalADT_DegradedRecommendsNativeEngine(t *testing.T) {
	t.Parallel()

	// Local degraded engine: O(N) but very fast per-op and zero RTT.
	degradedEng := &fakeEngine{profile: metaengine.EngineProfile{
		Name:      "fast-degraded",
		NsPerRead: 1,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTMap: true,
		},
	}}

	// Remote native engine: O(1) but high RTT makes it more expensive overall.
	nativeEng := &fakeEngine{profile: metaengine.EngineProfile{
		Name:       "slow-native",
		NsPerRead:  100,
		NetworkRTT: 10 * time.Millisecond,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan(
		[]metaengine.Engine{degradedEng, nativeEng},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// The planner should pick the fast-degraded engine (lower total latency).
	for _, q := range store.Plan().Queries {
		if q.ADT == metaengine.ADTMap && q.EngineName != "fast-degraded" {
			t.Fatalf("expected degraded engine selected, got %s", q.EngineName)
		}
	}

	// The degraded diagnostic should recommend the native engine.
	for _, d := range store.Plan().Diagnostics {
		if d.Level != metaengine.DiagLevelDegraded {
			continue
		}

		if !strings.Contains(d.Message, "native engine") {
			t.Errorf("expected native engine recommendation, got: %s", d.Message)
		}

		if !strings.Contains(d.Message, "slow-native") {
			t.Errorf("expected recommendation for 'slow-native', got: %s", d.Message)
		}

		return
	}

	t.Error("expected a DEGRADED diagnostic with native engine recommendation")
}

// TestUniversalADT_DegradedNoRecommendationWhenAlone verifies the diagnostic
// says "no native engine available" when only a degraded engine exists.
func TestUniversalADT_DegradedNoRecommendationWhenAlone(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "only-degraded",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTMap: true,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if d.Level != metaengine.DiagLevelDegraded {
			continue
		}

		if !strings.Contains(d.Message, "no native engine available") {
			t.Errorf("expected 'no native engine available', got: %s", d.Message)
		}

		return
	}

	t.Error("expected a DEGRADED diagnostic")
}

// TestDoctor_ShowsDegradedSection verifies Doctor() includes a "--- Degraded
// ADTs ---" section that lists queries on degraded engines.
func TestDoctor_ShowsDegradedSection(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "doctor-degraded",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTMap: true,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	report := store.Doctor(context.Background())

	if !strings.Contains(report, "--- Degraded ADTs ---") {
		t.Error("Doctor() should include a '--- Degraded ADTs ---' section")
	}

	if !strings.Contains(report, "find_task") {
		t.Error("Doctor() degraded section should list the degraded query")
	}

	if !strings.Contains(report, "doctor-degraded") {
		t.Error("Doctor() degraded section should name the degraded engine")
	}
}

// TestDoctor_NoDegradedSectionContentForNative verifies Doctor() shows "none"
// in the degraded section when all queries are on native engines.
func TestDoctor_NoDegradedSectionContentForNative(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "all-native",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	report := store.Doctor(context.Background())

	idx := strings.Index(report, "--- Degraded ADTs ---")
	if idx < 0 {
		t.Fatal("Doctor() should always include a '--- Degraded ADTs ---' section")
	}

	section := report[idx:]
	if !strings.Contains(section, "none") {
		t.Errorf("Degraded section should say 'none' for native-only, got:\n%s", section)
	}
}
