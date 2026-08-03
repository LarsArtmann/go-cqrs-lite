package metaengine_test

import (
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestReplicationRule_EmitsDiagnosticForReplicatedEngine(t *testing.T) {
	t.Parallel()

	lag := 50 * time.Millisecond
	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "replicated-pg",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationSingleLeader,
		ReplicationLag: lag,
		NetworkRTT:     5 * time.Millisecond,
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Query == "find_task" && d.Level == metaengine.DiagLevelInfo &&
			strings.Contains(d.Message, "single-leader") &&
			strings.Contains(d.Message, "50ms") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected replication INFO diagnostic for replicated engine, got: %v",
			store.Plan().Diagnostics)
	}
}

func TestReplicationRule_NoDiagnosticForLocalEngine(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "local-sqlite",
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
		if strings.Contains(d.Message, "replication") || strings.Contains(d.Message, "stale") {
			t.Errorf("local engine should not emit replication diagnostic: %s", d.Message)
		}
	}
}

func TestReplicationRule_NoDiagnosticWhenReplicatedButZeroLag(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "synced-cluster",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationMultiLeader,
		ReplicationLag: 0,
	}}

	store, err := metaengine.Plan([]metaengine.Engine{engine}, findTaskQuery())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for _, d := range store.Plan().Diagnostics {
		if strings.Contains(d.Message, "stale") {
			t.Errorf("replicated engine with zero lag should not emit staleness diagnostic: %s",
				d.Message)
		}
	}
}

func TestEngineProfileString_IncludesReplicationSuffix(t *testing.T) {
	t.Parallel()

	p := metaengine.EngineProfile{
		Name: "iroh-sync",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		Replication:    metaengine.ReplicationLeaderless,
		ReplicationLag: 200 * time.Millisecond,
		NetworkRTT:     5 * time.Millisecond,
	}

	s := p.String()
	for _, want := range []string{"iroh-sync", "replication=leaderless", "lag=200ms", "rtt=5ms"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() missing %q, got: %s", want, s)
		}
	}
}

func TestEngineProfileString_NoSuffixForLocalEngine(t *testing.T) {
	t.Parallel()

	p := metaengine.NewMemoryEngine().Profile()
	s := p.String()
	if strings.Contains(s, "replication=") {
		t.Errorf("local engine String() should not contain replication suffix, got: %s", s)
	}
}

func TestCollections_ExposesReplicationFields(t *testing.T) {
	t.Parallel()

	lag := 50 * time.Millisecond
	rtt := 5 * time.Millisecond
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

	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	c := collections[0]
	if c.Name != "find_task" {
		t.Errorf("expected collection 'find_task', got %q", c.Name)
	}
	if c.Replication != metaengine.ReplicationSingleLeader {
		t.Errorf("Replication: expected %q, got %q",
			metaengine.ReplicationSingleLeader, c.Replication)
	}
	if c.ReplicationLagMs != lag.Milliseconds() {
		t.Errorf("ReplicationLagMs: expected %d, got %d",
			lag.Milliseconds(), c.ReplicationLagMs)
	}
	if c.NetworkRTTMs != rtt.Milliseconds() {
		t.Errorf("NetworkRTTMs: expected %d, got %d",
			rtt.Milliseconds(), c.NetworkRTTMs)
	}
}

func TestCollections_ZeroReplicationForLocalEngine(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	c := collections[0]
	if c.Replication != metaengine.ReplicationNone {
		t.Errorf("local engine Replication: expected %q, got %q",
			metaengine.ReplicationNone, c.Replication)
	}
	if c.ReplicationLagMs != 0 {
		t.Errorf("local engine ReplicationLagMs: expected 0, got %d", c.ReplicationLagMs)
	}
	if c.NetworkRTTMs != 0 {
		t.Errorf("local engine NetworkRTTMs: expected 0, got %d", c.NetworkRTTMs)
	}
}
