package metaengine_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDurabilityRule_WARNForVolatileOnlyPlan(t *testing.T) {
	t.Parallel()

	// Only a volatile engine — no persistent alternative → WARN.
	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Query == "find_task" && d.Level == metaengine.DiagLevelWarn &&
			strings.Contains(d.Message, "volatile") &&
			strings.Contains(d.Message, "lost on restart") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected WARN diagnostic for volatile-only plan, got: %v",
			store.Plan().Diagnostics)
	}
}

func TestDurabilityRule_INFOWhenPersistentAlternativeExists(t *testing.T) {
	t.Parallel()

	// Two engines: volatile (Memory, O(1)) + persistent (fake, O(logN)).
	// Memory wins on cost → INFO showing the persistent alternative.
	volatileEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:        "volatile-cheap",
		NsPerOp:     100,
		Persistence: metaengine.PersistenceVolatile,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}
	persistentEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:        "persistent-slow",
		NsPerOp:     7000,
		Persistence: metaengine.PersistencePersistent,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityOLogN,
		},
	}}

	store, err := metaengine.Plan(
		[]metaengine.Engine{volatileEngine, persistentEngine},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Query == "find_task" && d.Level == metaengine.DiagLevelInfo &&
			strings.Contains(d.Message, "volatile") &&
			strings.Contains(d.Message, "persistent alternative") &&
			strings.Contains(d.Message, "ms/query") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected INFO diagnostic about persistent alternative with cost delta, got: %v",
			store.Plan().Diagnostics)
	}
}

func TestDurabilityRule_SilentWhenPersistentEngine(t *testing.T) {
	t.Parallel()

	persistentEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:        "persistent-only",
		NsPerOp:     5000,
		Persistence: metaengine.PersistencePersistent,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan(
		[]metaengine.Engine{persistentEngine},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if strings.Contains(d.Message, "volatile") ||
			strings.Contains(d.Message, "lost on restart") {
			t.Errorf("persistent engine should not emit durability diagnostic: %s", d.Message)
		}
	}
}

func TestDurabilityRule_RuleTraceEntry(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	found := false
	for _, rt := range store.Plan().RuleTrace {
		if rt.Rule == "durability" && rt.Query == "find_task" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected durability rule trace entry, got: %v", store.Plan().RuleTrace)
	}
}

func TestExplainPlan_ShowsVolatileForVolatileEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.ExplainPlan()
	if !strings.Contains(output, "volatile") {
		t.Errorf("ExplainPlan should show \"volatile\" for Memory engine, got:\n%s", output)
	}
}

func TestExplainPlan_NoVolatileForPersistentEngine(t *testing.T) {
	t.Parallel()

	persistentEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:        "persistent",
		NsPerOp:     5000,
		Persistence: metaengine.PersistencePersistent,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan(
		[]metaengine.Engine{persistentEngine},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.ExplainPlan()
	// The word "volatile" should not appear on the engine line for persistent engines.
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "persistent") && strings.Contains(line, "volatile") {
			t.Errorf("persistent engine line should not contain \"volatile\": %s", line)
		}
	}
}

func TestDoctor_ShowsPersistenceSectionForVolatileEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.Doctor(t.Context())
	for _, want := range []string{"--- Persistence ---", "find_task", "volatile"} {
		if !strings.Contains(output, want) {
			t.Errorf("Doctor: expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestDoctor_AllPersistentForPersistentEngine(t *testing.T) {
	t.Parallel()

	persistentEngine := &fakeEngine{profile: metaengine.EngineProfile{
		Name:        "persistent",
		NsPerOp:     5000,
		Persistence: metaengine.PersistencePersistent,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan(
		[]metaengine.Engine{persistentEngine},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	output := store.Doctor(t.Context())
	if !strings.Contains(output, "--- Persistence ---\n  all persistent") {
		t.Errorf("Doctor: persistent engine should show 'all persistent', got:\n%s", output)
	}
}
