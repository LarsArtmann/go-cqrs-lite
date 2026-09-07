package metaengine

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// lyingEngine declares ADTs natively without implementing any backend —
// the over-declaration the capability-aware router penalizes.
type lyingEngine struct {
	profile EngineProfile
}

func (e *lyingEngine) Profile() EngineProfile { return e.profile }
func (e *lyingEngine) Close() error           { return nil }

var _ Engine = (*lyingEngine)(nil)

func newLyingEngine(name string) *lyingEngine {
	return &lyingEngine{profile: EngineProfile{
		Name: name,
		Supports: map[ADT]Complexity{
			ADTMap: ComplexityO1,
		},
	}}
}

func capabilityQuery() QueryDecl[capabilityEvent, map[string]string] {
	return Query[capabilityEvent, map[string]string](
		"capability_tasks",
		OnRecord(capabilityEvent{}, func(_ record.Record, e capabilityEvent) (string, string) {
			return e.ID, e.ID
		}),
	)
}

// capabilityEvent is a local insert-fold event type for the router tests.
type capabilityEvent struct{ ID string }

// TestPlan_OverDeclaredEngineExcluded proves the routing penalty: when an
// engine declares ADTMap natively but implements no MapBackend, an honest
// engine wins the assignment and the plan carries a DEGRADED diagnostic.
func TestPlan_OverDeclaredEngineExcluded(t *testing.T) {
	t.Parallel()

	honest := NewMemoryEngine()
	store, err := Plan([]Engine{newLyingEngine("liar"), honest}, capabilityQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	assignment := store.Plan().Queries[0]
	if assignment.EngineName != honest.Profile().Name {
		t.Fatalf(
			"routed to %q, want honest engine %q",
			assignment.EngineName,
			honest.Profile().Name,
		)
	}

	found := false

	for _, d := range assignment.Diagnostics {
		if d.Level == DiagLevelDegraded && strings.Contains(d.Message, "over-declare ADT map") {
			found = true
		}
	}

	if !found {
		t.Fatalf("missing DEGRADED over-declaration diagnostic: %+v", assignment.Diagnostics)
	}
}

// TestPlan_OverDeclaredEngineWarnedWhenOnly proves the no-alternative case:
// with only an over-declaring engine, the query is still routed (a fallback
// path may serve it) and a WARN makes the execution-time hard-error risk
// visible at plan time instead of surfacing on first Apply.
func TestPlan_OverDeclaredEngineWarnedWhenOnly(t *testing.T) {
	t.Parallel()

	liar := newLyingEngine("liar")
	store, err := Plan([]Engine{liar}, capabilityQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	assignment := store.Plan().Queries[0]
	if assignment.EngineName != "liar" {
		t.Fatalf("routed to %q, want the only candidate %q", assignment.EngineName, "liar")
	}

	found := false

	for _, d := range assignment.Diagnostics {
		if d.Level == DiagLevelWarn && strings.Contains(d.Message, "over-declare ADT map") {
			found = true
		}
	}

	if !found {
		t.Fatalf("missing WARN over-declaration diagnostic: %+v", assignment.Diagnostics)
	}
}

// TestPlan_HonestEnginesNoCapabilityDiagnostics proves conforming engines
// plan exactly as before — no capability noise on the assignment.
func TestPlan_HonestEnginesNoCapabilityDiagnostics(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, capabilityQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	for _, d := range store.Plan().Queries[0].Diagnostics {
		if strings.Contains(d.Message, "over-declare") {
			t.Fatalf("unexpected over-declaration diagnostic: %+v", d)
		}
	}
}

// TestApply_CapabilityQuerySmoke applies one event through the planned store
// so the routing change is exercised end-to-end (fold dispatch unaffected).
func TestApply_CapabilityQuerySmoke(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, capabilityQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if err := store.Apply(
		context.Background(),
		"capabilityEvent",
		capabilityEvent{ID: "c1"},
	); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

// TestApply_LyingOnlyEngine_HardErrorCorrelatesWithPlanWarn correlates the
// two halves of the over-declaration contract: the plan-time WARN announces
// the execution-time risk, and the runtime actually hard-errors on Apply
// instead of silently no-oping. The error must carry the ApplyError structure
// and name the unsupported capability.
func TestApply_LyingOnlyEngine_HardErrorCorrelatesWithPlanWarn(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{newLyingEngine("liar")}, capabilityQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	warned := false

	for _, d := range store.Plan().Queries[0].Diagnostics {
		if d.Level == DiagLevelWarn && strings.Contains(d.Message, "over-declare") {
			warned = true
		}
	}

	if !warned {
		t.Fatalf("plan must WARN about the over-declaration before Apply:\n%+v",
			store.Plan().Queries[0].Diagnostics)
	}

	applyErr := store.Apply(context.Background(), "capabilityEvent", capabilityEvent{ID: "c1"})
	if applyErr == nil {
		t.Fatal("Apply on a lying-only-engine store must hard-error, got nil")
	}

	var applyErrStruct *ApplyError
	if !errors.As(applyErr, &applyErrStruct) {
		t.Fatalf("Apply error must wrap *ApplyError, got %T: %v", applyErr, applyErr)
	}

	if !strings.Contains(applyErr.Error(), "liar") {
		t.Fatalf("Apply error must name the engine, got: %v", applyErr)
	}
}

// TestReplan_KeepsExcludingOverDeclaredEngine proves the capability partition
// is stable across re-plans: a Replan (e.g. triggered by a live-latency shift)
// must not re-route the query onto an engine that declares the ADT without
// implementing the backend.
func TestReplan_KeepsExcludingOverDeclaredEngine(t *testing.T) {
	t.Parallel()

	honest := NewMemoryEngine()

	store, err := Plan([]Engine{newLyingEngine("liar"), honest}, capabilityQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if err := store.Replan(context.Background()); err != nil {
		t.Fatalf("Replan: %v", err)
	}

	if got := store.Plan().Queries[0].EngineName; got != honest.Profile().Name {
		t.Fatalf("after Replan routed to %q, want honest engine %q",
			got, honest.Profile().Name)
	}
}

// TestCheckRouting_NeverSuggestsOverDeclaredEngine pins the routing-check
// half of the partition: a lying engine with cheaper priors looks strictly
// better than the honest incumbent, but CheckRouting must not emit a
// REPLAN-SUGGESTED diagnostic pointing at an engine Replan would refuse and
// whose Apply would hard-error.
func TestCheckRouting_NeverSuggestsOverDeclaredEngine(t *testing.T) {
	t.Parallel()

	liar := &lyingEngine{profile: EngineProfile{
		Name:      "liar",
		Supports:  map[ADT]Complexity{ADTMap: ComplexityO1},
		ReadCosts: ReadCosts{NsPerPointLookup: 1},
	}}

	honest := NewMemoryEngine()

	store, err := Plan([]Engine{liar, honest}, capabilityQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	diags := store.CheckRouting(context.Background())

	for _, d := range diags {
		if strings.Contains(d.Message, "liar") {
			t.Fatalf("CheckRouting suggested the over-declaring engine: %+v", d)
		}
	}
}
