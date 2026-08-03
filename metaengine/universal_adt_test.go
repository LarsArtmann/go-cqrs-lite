package metaengine_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// allADTs returns the canonical 10 ADTs.
func allADTs() []metaengine.ADT {
	return []metaengine.ADT{
		metaengine.ADTMap,
		metaengine.ADTSet,
		metaengine.ADTCounter,
		metaengine.ADTGraph,
		metaengine.ADTSortedMap,
		metaengine.ADTLog,
		metaengine.ADTMultimap,
		metaengine.ADTVector,
		metaengine.ADTSearch,
		metaengine.ADTSpatial,
	}
}

// TestUniversalADT_MemoryEngineHasAllTenADTs verifies that the Memory engine
// declares support for all 10 ADTs — the baseline for universal ADT coverage.
func TestUniversalADT_MemoryEngineHasAllTenADTs(t *testing.T) {
	t.Parallel()

	profile := metaengine.NewMemoryEngine().Profile()
	for _, adt := range allADTs() {
		if _, ok := profile.Supports[adt]; !ok {
			t.Errorf("Memory engine missing ADT %s in Supports map", adt)
		}
		if profile.IsDegraded(adt) {
			t.Errorf("Memory engine should not mark %s as degraded (all native)", adt)
		}
	}
}

// TestUniversalADT_DegradedDiagnosticEmitted verifies that a DEGRADED
// diagnostic is emitted when a query routes to an engine via a degraded
// fallback (ADT in DegradedADTs).
func TestUniversalADT_DegradedDiagnosticEmitted(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "test-degraded-map",
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

	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Level == metaengine.DiagLevelDegraded &&
			strings.Contains(d.Message, "DEGRADED") &&
			strings.Contains(d.Message, "map") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected DEGRADED diagnostic for map on degraded engine, got: %v",
			store.Plan().Diagnostics)
	}
}

// TestUniversalADT_NoDegradedDiagnosticForNative verifies that NO degraded
// diagnostic is emitted when the engine handles the ADT natively.
func TestUniversalADT_NoDegradedDiagnosticForNative(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "test-native-map",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if d.Level == metaengine.DiagLevelDegraded {
			t.Errorf("native ADT should not emit DEGRADED diagnostic: %s", d.Message)
		}
	}
}

// TestUniversalADT_PrefersNativeOverDegraded verifies that when two engines
// are available — one native and one degraded — the planner picks the native
// one and does NOT emit a degraded diagnostic.
func TestUniversalADT_PrefersNativeOverDegraded(t *testing.T) {
	t.Parallel()

	nativeEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "native-memory",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}
	degradedEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "degraded-sqlite",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTMap: true,
		},
	}}

	store, err := metaengine.Plan(
		[]metaengine.Engine{nativeEngine, degradedEngine},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, q := range store.Plan().Queries {
		if q.ADT == metaengine.ADTMap && q.EngineName != "native-memory" {
			t.Errorf("expected native engine, got %s", q.EngineName)
		}
	}

	for _, d := range store.Plan().Diagnostics {
		if d.Level == metaengine.DiagLevelDegraded {
			t.Errorf("native routing should not emit DEGRADED: %s", d.Message)
		}
	}
}

// TestUniversalADT_DegradedRuleInRuleTrace verifies the degraded-adt rule
// appears in RuleTrace when a degraded diagnostic fires.
func TestUniversalADT_DegradedRuleInRuleTrace(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "trace-degraded",
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

	found := false
	for _, rt := range store.Plan().RuleTrace {
		if rt.Rule == "degraded-adt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected degraded-adt in RuleTrace, got: %v", store.Plan().RuleTrace)
	}
}

// TestUniversalADT_NoDegradedWhenDegradedADTsNil verifies that an engine
// with nil DegradedADTs never triggers a degraded diagnostic, even with
// all 10 ADTs in Supports.
func TestUniversalADT_NoDegradedWhenDegradedADTsNil(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "clean-engine",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityO1,
			metaengine.ADTSet:       metaengine.ComplexityO1,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTGraph:     metaengine.ComplexityODegree,
			metaengine.ADTSortedMap: metaengine.ComplexityON,
			metaengine.ADTLog:       metaengine.ComplexityON,
			metaengine.ADTMultimap:  metaengine.ComplexityO1,
			metaengine.ADTVector:    metaengine.ComplexityON,
			metaengine.ADTSearch:    metaengine.ComplexityON,
			metaengine.ADTSpatial:   metaengine.ComplexityON,
		},
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if d.Level == metaengine.DiagLevelDegraded {
			t.Errorf("engine with nil DegradedADTs should not emit DEGRADED: %s", d.Message)
		}
	}
}

// TestUniversalADT_IsDegradedMethod verifies the IsDegraded method on
// EngineProfile works correctly for both set and unset cases.
func TestUniversalADT_IsDegradedMethod(t *testing.T) {
	t.Parallel()

	withDegraded := metaengine.EngineProfile{
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:    metaengine.ComplexityO1,
			metaengine.ADTVector: metaengine.ComplexityON,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTVector: true,
		},
	}

	if withDegraded.IsDegraded(metaengine.ADTVector) != true {
		t.Error("ADTVector should be degraded")
	}
	if withDegraded.IsDegraded(metaengine.ADTMap) != false {
		t.Error("ADTMap should not be degraded")
	}

	withoutDegraded := metaengine.EngineProfile{
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
	if withoutDegraded.IsDegraded(metaengine.ADTMap) {
		t.Error("nil DegradedADTs should return false for all ADTs")
	}
}
