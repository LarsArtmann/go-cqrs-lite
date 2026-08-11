package system_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const o1 = string(metaengine.ComplexityO1)

// ─── Scream Store Plan Safety Tests ───

func testPlan(queries ...metaengine.SerializableQuery) *metaengine.SerializablePlan {
	return &metaengine.SerializablePlan{
		Engines: []string{"memory"},
		Queries: queries,
	}
}

func TestCheckPlanSafety_FirstDeployment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plan.json")
	plan := testPlan(metaengine.SerializableQuery{
		Name: "counts", ADT: metaengine.ADTCounter, Engine: "memory",
		Complexity: o1,
	})

	report, err := system.CheckPlanSafety(context.Background(), plan, manifestPath)
	if err != nil {
		t.Fatalf("CheckPlanSafety: %v", err)
	}

	if report.HasErrors() {
		t.Fatalf("first deployment should not have SCREAMs, got: %+v", report.Diagnostics)
	}

	found := false
	for _, d := range report.Diagnostics {
		if d.Rule == "plan:first-deployment" {
			found = true
			if d.Tier != system.TierAdvisory {
				t.Fatalf("first deployment should be ADVISORY, got %s", d.Tier)
			}
		}
	}

	if !found {
		t.Fatal("expected plan:first-deployment diagnostic")
	}

	// Manifest should have been saved.
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest file should exist after first deployment: %v", err)
	}
}

func TestCheckPlanSafety_NoDrift(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plan.json")
	queries := []metaengine.SerializableQuery{
		{Name: "counts", ADT: metaengine.ADTCounter, Engine: "memory", Complexity: o1},
		{Name: "items", ADT: metaengine.ADTMap, Engine: "memory", Complexity: o1},
	}
	plan := testPlan(queries...)

	// First deployment — saves manifest.
	_, err := system.CheckPlanSafety(context.Background(), plan, manifestPath)
	if err != nil {
		t.Fatalf("first deployment: %v", err)
	}

	// Second deployment with identical plan — no diff.
	report, err := system.CheckPlanSafety(context.Background(), testPlan(queries...), manifestPath)
	if err != nil {
		t.Fatalf("second deployment: %v", err)
	}

	if len(report.Diagnostics) != 0 {
		t.Fatalf("expected 0 diagnostics on identical plan, got: %+v", report.Diagnostics)
	}
}

func TestCheckPlanSafety_QueryRemoved_SCREAM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plan.json")
	queries := []metaengine.SerializableQuery{
		{Name: "counts", ADT: metaengine.ADTCounter, Engine: "memory", Complexity: o1},
		{Name: "items", ADT: metaengine.ADTMap, Engine: "memory", Complexity: o1},
	}

	// First deployment.
	_, err := system.CheckPlanSafety(context.Background(), testPlan(queries...), manifestPath)
	if err != nil {
		t.Fatalf("first deployment: %v", err)
	}

	// Second deployment with "items" removed.
	current := testPlan(queries[0])
	report, err := system.CheckPlanSafety(context.Background(), current, manifestPath)
	if err != nil {
		t.Fatalf("second deployment: %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected SCREAM when query removed")
	}

	found := false
	for _, d := range report.Diagnostics {
		if d.Rule == "plan:query-removed:items" && d.Tier == system.TierScream {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected plan:query-removed:items SCREAM, got: %+v", report.Diagnostics)
	}
}

func TestCheckPlanSafety_ADTChanged_SCREAM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plan.json")

	old := testPlan(metaengine.SerializableQuery{
		Name: "counts", ADT: metaengine.ADTCounter, Engine: "memory", Complexity: o1,
	})

	_, err := system.CheckPlanSafety(context.Background(), old, manifestPath)
	if err != nil {
		t.Fatalf("first deployment: %v", err)
	}

	// Change ADT from Counter to Map.
	current := testPlan(metaengine.SerializableQuery{
		Name: "counts", ADT: metaengine.ADTMap, Engine: "memory", Complexity: o1,
	})

	report, err := system.CheckPlanSafety(context.Background(), current, manifestPath)
	if err != nil {
		t.Fatalf("second deployment: %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected SCREAM when ADT changed")
	}

	found := false
	for _, d := range report.Diagnostics {
		if d.Rule == "plan:adt-changed:counts" && d.Tier == system.TierScream {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected plan:adt-changed:counts SCREAM, got: %+v", report.Diagnostics)
	}
}

func TestCheckPlanSafety_EngineChanged_WARN(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plan.json")

	old := &metaengine.SerializablePlan{
		Engines: []string{"memory", "sqlite"},
		Queries: []metaengine.SerializableQuery{
			{Name: "counts", ADT: metaengine.ADTCounter, Engine: "memory", Complexity: o1},
		},
	}

	_, err := system.CheckPlanSafety(context.Background(), old, manifestPath)
	if err != nil {
		t.Fatalf("first deployment: %v", err)
	}

	// Change engine from memory to sqlite (same ADT).
	current := &metaengine.SerializablePlan{
		Engines: []string{"memory", "sqlite"},
		Queries: []metaengine.SerializableQuery{
			{Name: "counts", ADT: metaengine.ADTCounter, Engine: "sqlite", Complexity: o1},
		},
	}

	report, err := system.CheckPlanSafety(context.Background(), current, manifestPath)
	if err != nil {
		t.Fatalf("second deployment: %v", err)
	}

	if report.HasErrors() {
		t.Fatalf("engine change should be WARN, not SCREAM: %+v", report.Diagnostics)
	}

	found := false
	for _, d := range report.Diagnostics {
		if d.Rule == "plan:engine-changed:counts" && d.Tier == system.TierWarnOverride {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected plan:engine-changed:counts WARN, got: %+v", report.Diagnostics)
	}
}

func TestCheckPlanSafety_QueryAdded_ADVISORY(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plan.json")

	old := testPlan(metaengine.SerializableQuery{
		Name: "counts", ADT: metaengine.ADTCounter, Engine: "memory", Complexity: o1,
	})

	_, err := system.CheckPlanSafety(context.Background(), old, manifestPath)
	if err != nil {
		t.Fatalf("first deployment: %v", err)
	}

	// Add a new query.
	current := testPlan(
		metaengine.SerializableQuery{
			Name:       "counts",
			ADT:        metaengine.ADTCounter,
			Engine:     "memory",
			Complexity: o1,
		},
		metaengine.SerializableQuery{
			Name:       "items",
			ADT:        metaengine.ADTMap,
			Engine:     "memory",
			Complexity: o1,
		},
	)

	report, err := system.CheckPlanSafety(context.Background(), current, manifestPath)
	if err != nil {
		t.Fatalf("second deployment: %v", err)
	}

	if report.HasErrors() {
		t.Fatalf("added query should not SCREAM: %+v", report.Diagnostics)
	}

	found := false
	for _, d := range report.Diagnostics {
		if d.Rule == "plan:query-added:items" && d.Tier == system.TierAdvisory {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected plan:query-added:items ADVISORY, got: %+v", report.Diagnostics)
	}
}

func TestCheckPlanSafety_EmptyManifestPath_NoOp(t *testing.T) {
	t.Parallel()

	plan := testPlan(metaengine.SerializableQuery{
		Name: "counts", ADT: metaengine.ADTCounter, Engine: "memory",
	})

	report, err := system.CheckPlanSafety(context.Background(), plan, "")
	if err != nil {
		t.Fatalf("CheckPlanSafety: %v", err)
	}

	if len(report.Diagnostics) != 0 {
		t.Fatalf("empty manifest path should produce no diagnostics, got: %+v", report.Diagnostics)
	}
}

func TestCheckPlanSafety_NilPlan_NoOp(t *testing.T) {
	t.Parallel()

	report, err := system.CheckPlanSafety(context.Background(), nil, "/tmp/whatever.json")
	if err != nil {
		t.Fatalf("CheckPlanSafety: %v", err)
	}

	if len(report.Diagnostics) != 0 {
		t.Fatalf("nil plan should produce no diagnostics, got: %+v", report.Diagnostics)
	}
}

// ─── System integration: manifest blocks on SCREAM ───

// taskCountInput is a minimal input type for metaengine Query declarations.
type taskCountInput struct{}

func TestSystem_ManifestPath_BlocksOnRemovedQuery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plan.json")

	// First deployment — pins a plan with two projections.
	proj1 := metaengine.Query[taskCountInput, map[string]int64]("counts",
		metaengine.OnRecord(TaskCreated{}, func(_ record.Record, e TaskCreated) metaengine.Delta {
			return metaengine.Delta{"pending": +1}
		}),
	)
	proj2 := metaengine.Query[taskCountInput, map[string]int64]("items",
		metaengine.OnRecord(TaskCreated{}, func(_ record.Record, e TaskCreated) metaengine.Delta {
			return metaengine.Delta{"pending": +1}
		}),
	)

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
		ManifestPath: manifestPath,
	}

	domain1 := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{system.RawQuery(proj1), system.RawQuery(proj2)},
	}

	sys1, err := system.New(context.Background(), domain1, deployment)
	if err != nil {
		t.Fatalf("first system.New: %v", err)
	}

	sys1.Close()

	// Second deployment — removes proj2 ("items"), should SCREAM.
	domain2 := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{system.RawQuery(proj1)},
	}

	_, err = system.New(context.Background(), domain2, deployment)
	if err == nil {
		t.Fatal("expected error when query removed but manifest pinned")
	}
}

// ─── CommandAdapter Serialization Tests ───

func TestCommandAdapter_SerializationRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := metaengine.NewMemoryEngine()
	adapter := system.NewCommandAdapter(eng.(metaengine.StreamLogBackend), "commands",
		system.WithCommandSerialization())

	streamID := id.NewStreamID()
	ref := command.NewStreamRef("Task", streamID)

	original, err := command.NewPersistedCommand(
		"task.create", ref, []byte(`{"title":"test"}`),
		command.WithCommandMetadata(command.Metadata{
			Custom: map[command.MetadataKey]string{"source": "api"},
		}),
	)
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := adapter.Save(ctx, ref, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := adapter.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}

	cmd := loaded[0]
	if cmd.ID() != original.ID() {
		t.Fatalf("ID mismatch: %s != %s", cmd.ID(), original.ID())
	}

	if cmd.Type() != original.Type() {
		t.Fatalf("Type mismatch: %s != %s", cmd.Type(), original.Type())
	}

	if cmd.StreamID() != original.StreamID() {
		t.Fatalf("StreamID mismatch: %s != %s", cmd.StreamID(), original.StreamID())
	}

	if cmd.StreamType() != original.StreamType() {
		t.Fatalf("StreamType mismatch: %s != %s", cmd.StreamType(), original.StreamType())
	}

	if !cmd.ReceivedAt().Equal(original.ReceivedAt()) {
		t.Fatalf("ReceivedAt mismatch: %v != %v", cmd.ReceivedAt(), original.ReceivedAt())
	}

	if string(cmd.Payload()) != string(original.Payload()) {
		t.Fatalf("Payload mismatch: %s != %s", cmd.Payload(), original.Payload())
	}

	if cmd.Metadata().Custom["source"] != "api" {
		t.Fatalf("Metadata custom mismatch: %v", cmd.Metadata().Custom)
	}
}

func TestCommandAdapter_MemoryMode_NoSerialization(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := metaengine.NewMemoryEngine()
	// No serialization option — direct pointer mode.
	adapter := system.NewCommandAdapter(eng.(metaengine.StreamLogBackend), "commands")

	streamID := id.NewStreamID()
	ref := command.NewStreamRef("Task", streamID)

	cmd, err := command.NewPersistedCommand("task.create", ref, []byte(`{}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := adapter.Save(ctx, ref, cmd); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := adapter.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 command, got %d", len(loaded))
	}

	if loaded[0].ID() != cmd.ID() {
		t.Fatalf("ID mismatch in memory mode")
	}
}

func TestCommandAdapter_ReadAllAndReadFrom_Serialized(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := metaengine.NewMemoryEngine()
	adapter := system.NewCommandAdapter(eng.(metaengine.StreamLogBackend), "commands",
		system.WithCommandSerialization())

	ref := command.NewStreamRef("Task", id.NewStreamID())

	for i := 0; i < 3; i++ {
		cmd, err := command.NewPersistedCommand("task.create", ref, []byte(`{}`))
		if err != nil {
			t.Fatalf("NewPersistedCommand: %v", err)
		}

		if err := adapter.Save(ctx, ref, cmd); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	// ReadAll
	all, err := adapter.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(all))
	}

	// ReadFrom after the first command
	from, err := adapter.ReadFrom(ctx, all[0].ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(from) != 2 {
		t.Fatalf("expected 2 commands after first, got %d", len(from))
	}
}

// ─── QueryAdapter Serialization Tests ───

func TestQueryAdapter_SerializationRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := metaengine.NewMemoryEngine()
	adapter := system.NewQueryAdapter(eng.(metaengine.StreamLogBackend), "queries",
		system.WithQuerySerialization())

	original, err := query.NewPersistedQuery(
		"task.get", []byte(`{"id":"123"}`),
		query.WithQueryMetadata(query.Metadata{
			Custom: map[query.MetadataKey]string{"source": "api"},
		}),
	)
	if err != nil {
		t.Fatalf("NewPersistedQuery: %v", err)
	}

	if err := adapter.SaveQuery(ctx, original); err != nil {
		t.Fatalf("SaveQuery: %v", err)
	}

	// LoadQueries with a time before receivedAt — should return all.
	loaded, err := adapter.LoadQueries(ctx, original.ReceivedAt().Add(-time.Hour))
	if err != nil {
		t.Fatalf("LoadQueries: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 query, got %d", len(loaded))
	}

	q := loaded[0]
	if q.ID() != original.ID() {
		t.Fatalf("ID mismatch: %s != %s", q.ID(), original.ID())
	}

	if q.Type() != original.Type() {
		t.Fatalf("Type mismatch: %s != %s", q.Type(), original.Type())
	}

	if !q.ReceivedAt().Equal(original.ReceivedAt()) {
		t.Fatalf("ReceivedAt mismatch")
	}

	if string(q.Payload()) != string(original.Payload()) {
		t.Fatalf("Payload mismatch: %s != %s", q.Payload(), original.Payload())
	}

	if q.Metadata().Custom["source"] != "api" {
		t.Fatalf("Metadata custom mismatch: %v", q.Metadata().Custom)
	}
}

func TestQueryAdapter_ReadAllQueriesAndReadQueriesFrom_Serialized(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eng := metaengine.NewMemoryEngine()
	adapter := system.NewQueryAdapter(eng.(metaengine.StreamLogBackend), "queries",
		system.WithQuerySerialization())

	for i := 0; i < 3; i++ {
		q, err := query.NewPersistedQuery("task.get", []byte(`{}`))
		if err != nil {
			t.Fatalf("NewPersistedQuery: %v", err)
		}

		if err := adapter.SaveQuery(ctx, q); err != nil {
			t.Fatalf("SaveQuery: %v", err)
		}
	}

	all, err := adapter.ReadAllQueries(ctx)
	if err != nil {
		t.Fatalf("ReadAllQueries: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 queries, got %d", len(all))
	}

	// ReadQueriesFrom after the first query.
	from, err := adapter.ReadQueriesFrom(ctx, all[0].ID(), 10)
	if err != nil {
		t.Fatalf("ReadQueriesFrom: %v", err)
	}

	if len(from) != 2 {
		t.Fatalf("expected 2 queries after first, got %d", len(from))
	}
}
