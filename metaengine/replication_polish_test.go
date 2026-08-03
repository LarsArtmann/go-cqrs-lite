package metaengine_test

import (
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── T14-T15: WithReplication / WithNetworkRTT plan options ───

func TestWithNetworkRTT_IncreasesCostEstimate(t *testing.T) {
	t.Parallel()

	// Same engine planned with and without network RTT override.
	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "remote-engine",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		NsPerOp: 1000,
	}}

	storeBase, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer storeBase.Close()

	storeWithRTT, err := metaengine.Plan(
		[]metaengine.Engine{engine},
		metaengine.WithNetworkRTT(50*time.Millisecond),
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer storeWithRTT.Close()

	baseLatency := storeBase.Plan().Queries[0].Cost.EstimatedLatencyMs
	rttLatency := storeWithRTT.Plan().Queries[0].Cost.EstimatedLatencyMs

	if rttLatency <= baseLatency {
		t.Errorf("WithNetworkRTT should increase cost: base=%.3fms, withRTT=%.3fms",
			baseLatency, rttLatency)
	}

	// 50ms RTT should add ~50ms to the estimate.
	if rttLatency-baseLatency < 49 {
		t.Errorf("expected ~50ms increase, got %.3fms (base=%.3f, rtt=%.3f)",
			rttLatency-baseLatency, baseLatency, rttLatency)
	}
}

func TestWithReplication_DoesNotError(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "local-engine",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	store, err := metaengine.Plan(
		[]metaengine.Engine{engine},
		metaengine.WithReplication(metaengine.ReplicationSingleLeader),
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
}

// ─── T16: SerializableQuery replication fields ───

func TestSerializablePlan_IncludesReplicationFields(t *testing.T) {
	t.Parallel()

	lag := 100 * time.Millisecond
	rtt := 10 * time.Millisecond
	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "replicated-pg",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationSingleLeader,
		ReplicationLag: lag,
		NetworkRTT:     rtt,
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sp := metaengine.Serialize(store.Plan(), []metaengine.Engine{engine})
	if len(sp.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(sp.Queries))
	}

	q := sp.Queries[0]
	if q.Replication != metaengine.ReplicationSingleLeader {
		t.Errorf("Replication: expected %q, got %q",
			metaengine.ReplicationSingleLeader, q.Replication)
	}

	if q.ReplicationLagMs != lag.Milliseconds() {
		t.Errorf("ReplicationLagMs: expected %d, got %d",
			lag.Milliseconds(), q.ReplicationLagMs)
	}

	if q.NetworkRTTMs != rtt.Milliseconds() {
		t.Errorf("NetworkRTTMs: expected %d, got %d",
			rtt.Milliseconds(), q.NetworkRTTMs)
	}
}

func TestSerializablePlan_ZeroReplicationForLocalEngine(t *testing.T) {
	t.Parallel()

	engine := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sp := metaengine.Serialize(store.Plan(), []metaengine.Engine{engine})
	q := sp.Queries[0]

	if q.Replication != metaengine.ReplicationNone {
		t.Errorf("local engine Replication: expected %q, got %q",
			metaengine.ReplicationNone, q.Replication)
	}

	if q.ReplicationLagMs != 0 {
		t.Errorf("local engine ReplicationLagMs: expected 0, got %d", q.ReplicationLagMs)
	}

	if q.NetworkRTTMs != 0 {
		t.Errorf("local engine NetworkRTTMs: expected 0, got %d", q.NetworkRTTMs)
	}
}

// ─── T17: ReplicationMode accessor ───

func TestReplicationMode_ReturnsCorrectMode(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "cockroach",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication: metaengine.ReplicationLeaderless,
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if mode := store.ReplicationMode("find_task"); mode != metaengine.ReplicationLeaderless {
		t.Errorf("expected %q, got %q", metaengine.ReplicationLeaderless, mode)
	}
}

func TestReplicationMode_NoneForLocalEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if mode := store.ReplicationMode("find_task"); mode != metaengine.ReplicationNone {
		t.Errorf("local engine: expected %q, got %q", metaengine.ReplicationNone, mode)
	}
}

func TestReplicationMode_NoneForUnknownQuery(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if mode := store.ReplicationMode("nonexistent"); mode != metaengine.ReplicationNone {
		t.Errorf("unknown query: expected %q, got %q", metaengine.ReplicationNone, mode)
	}
}

// ─── T18: MapUpdate WARN diagnostic ───

func TestMapUpdateReplicationRule_EmitsWarn(t *testing.T) {
	t.Parallel()

	lag := 200 * time.Millisecond
	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "multi-leader-pg",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationMultiLeader,
		ReplicationLag: lag,
		NetworkRTT:     5 * time.Millisecond,
	}}

	// findTaskQuery has a TaskCompleted fold which is FoldUpdate (takes prev FindTaskResult).
	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Query == "find_task" && d.Level == metaengine.DiagLevelWarn &&
			strings.Contains(d.Message, "MapUpdate") &&
			strings.Contains(d.Message, "multi-leader") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf(
			"expected MapUpdate WARN diagnostic for replicated engine with update folds, got: %v",
			store.Plan().Diagnostics,
		)
	}
}

func TestMapUpdateReplicationRule_NoWarnForLocalEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if strings.Contains(d.Message, "MapUpdate") {
			t.Errorf("local engine should not emit MapUpdate warning: %s", d.Message)
		}
	}
}

func TestMapUpdateReplicationRule_NoWarnForReplicatedZeroLag(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "synced-cluster",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationSingleLeader,
		ReplicationLag: 0,
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if strings.Contains(d.Message, "MapUpdate") {
			t.Errorf("zero-lag replicated engine should not emit MapUpdate warning: %s", d.Message)
		}
	}
}

func TestMapUpdateReplicationRule_NoWarnForInsertOnlyMap(t *testing.T) {
	t.Parallel()

	// A Map query with only insert folds (no update) should not trigger the warning.
	type simpleInput struct{ ID string }
	type simpleResult struct{ Name string }

	type itemAdded struct {
		ID   string
		Name string
	}

	q := metaengine.Query[simpleInput, simpleResult](
		"simple_items",
		metaengine.On(itemAdded{}, func(e itemAdded) (string, simpleResult) {
			return e.ID, simpleResult{Name: e.Name}
		}),
	)

	lag := 100 * time.Millisecond
	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "replicated-store",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationLeaderless,
		ReplicationLag: lag,
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if strings.Contains(d.Message, "MapUpdate") {
			t.Errorf("insert-only Map query should not emit MapUpdate warning: %s", d.Message)
		}
	}
}

func TestMapUpdateReplicationRule_InRuleTrace(t *testing.T) {
	t.Parallel()

	lag := 50 * time.Millisecond
	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "replicated-store",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationSingleLeader,
		ReplicationLag: lag,
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	found := false
	for _, rt := range store.Plan().RuleTrace {
		if rt.Rule == "mapupdate-replication" && rt.Query == "find_task" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected mapupdate-replication in RuleTrace, got: %v",
			store.Plan().RuleTrace)
	}
}
