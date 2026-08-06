package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestSerializableReadCosts_RoundTrip(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "calibrated-sqlite",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
		ReadCosts: metaengine.ReadCosts{
			NsPerPointLookup:  7000,
			NsPerFilteredScan: 50000,
			NsPerAggregate:    200000,
			NsPerScan:         1500,
		},
	}}

	type input struct{}
	type result map[TaskID]string

	decl := metaengine.Query[input, result]("test-map",
		metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, result) {
			return e.ID, result{e.ID: e.Title}
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{engine}, decl)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	sp := metaengine.Serialize(store.Plan(), []metaengine.Engine{engine})

	if len(sp.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(sp.Queries))
	}

	sq := sp.Queries[0]
	if sq.ReadCosts == nil {
		t.Fatal("expected ReadCosts to be non-nil for calibrated engine")
	}

	want := metaengine.SerializableReadCosts{
		NsPerPointLookup:  7000,
		NsPerFilteredScan: 50000,
		NsPerAggregate:    200000,
		NsPerScan:         1500,
	}
	if *sq.ReadCosts != want {
		t.Errorf("ReadCosts mismatch:\n  got:  %+v\n  want: %+v", *sq.ReadCosts, want)
	}

	data, err := sp.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	deserialized, err := metaengine.DeserializePlan(data)
	if err != nil {
		t.Fatalf("DeserializePlan failed: %v", err)
	}

	if len(deserialized.Queries) != 1 {
		t.Fatalf("expected 1 query after deserialize, got %d", len(deserialized.Queries))
	}

	dq := deserialized.Queries[0]
	if dq.ReadCosts == nil {
		t.Fatal("expected ReadCosts to survive JSON round-trip")
	}

	if *dq.ReadCosts != want {
		t.Errorf(
			"ReadCosts mismatch after round-trip:\n  got:  %+v\n  want: %+v",
			*dq.ReadCosts,
			want,
		)
	}
}

func TestSerializableReadCosts_NilWhenUncalibrated(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{profile: metaengine.EngineProfile{
		Name: "uncalibrated-memory",
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}}

	type input struct{}
	type result map[TaskID]string

	decl := metaengine.Query[input, result]("test-map",
		metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, result) {
			return e.ID, result{e.ID: e.Title}
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{engine}, decl)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	sp := metaengine.Serialize(store.Plan(), []metaengine.Engine{engine})

	if len(sp.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(sp.Queries))
	}

	if sp.Queries[0].ReadCosts != nil {
		t.Errorf(
			"expected ReadCosts to be nil for uncalibrated engine, got %+v",
			sp.Queries[0].ReadCosts,
		)
	}

	data, err := sp.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	deserialized, err := metaengine.DeserializePlan(data)
	if err != nil {
		t.Fatalf("DeserializePlan failed: %v", err)
	}

	if deserialized.Queries[0].ReadCosts != nil {
		t.Error("expected ReadCosts to remain nil after round-trip for uncalibrated engine")
	}
}

func TestSerializableReadCosts_MemoryEngineSerializes(t *testing.T) {
	t.Parallel()

	memEngine := metaengine.NewMemoryEngine()

	type input struct{}
	type result map[TaskID]string

	decl := metaengine.Query[input, result]("counts",
		metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, result) {
			return e.ID, result{e.ID: e.Title}
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{memEngine}, decl)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	sp := metaengine.Serialize(store.Plan(), []metaengine.Engine{memEngine})
	if len(sp.Queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(sp.Queries))
	}

	data, err := sp.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	if _, err := metaengine.DeserializePlan(data); err != nil {
		t.Fatalf("DeserializePlan failed: %v", err)
	}
}
