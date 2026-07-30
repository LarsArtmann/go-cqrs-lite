package correctness_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

// C017: In-memory snapshot store with persistent event store fires.
func TestC017_InMemSnapshotWithPersistentStore(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := memory.NewMemorySnapshotStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings, err := correctness.NewC017Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	if string(findings[0].Rule) != "C017" {
		t.Errorf("expected C017, got %s", findings[0].Rule)
	}
}

// C017: No finding when store is also in-memory.
func TestC017_NoFindingForAllMemory(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := memory.NewMemorySnapshotStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreMemory

	findings, err := correctness.NewC017Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for all-memory, got %d", len(findings))
	}
}

// C017: No finding when persistent store uses persistent snapshot.
func TestC017_NoFindingForPersistentSnapshot(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := storage.NewSQLSnapshotStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings, err := correctness.NewC017Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

// C017: In-memory checkpoint store with persistent event store fires (A030).
func TestC017_InMemCheckpointStoreWithPersistentStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := memory.NewMemoryCheckpointStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings, err := correctness.NewC017Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for in-memory checkpoint store, got %d", len(findings))
	}

	if string(findings[0].Rule) != "C017" {
		t.Errorf("expected C017, got %s", findings[0].Rule)
	}
}

// C017: In-memory dead-letter store from projectionhost with persistent event
// store fires (A031).
func TestC017_InMemDeadLetterStoreWithPersistentStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := projectionhost.NewMemoryDeadLetterStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings, err := correctness.NewC017Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for in-memory DLQ, got %d", len(findings))
	}

	if string(findings[0].Rule) != "C017" {
		t.Errorf("expected C017, got %s", findings[0].Rule)
	}
}

// C017: In-memory timer store with persistent event store fires.
func TestC017_InMemTimerStoreWithPersistentStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	store := scheduling.NewMemoryTimerStore()
	_ = store
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePostgres

	findings, err := correctness.NewC017Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for in-memory timer store, got %d", len(findings))
	}

	if string(findings[0].Rule) != "C017" {
		t.Errorf("expected C017, got %s", findings[0].Rule)
	}
}

// C017: No finding when the same file uses memory.NewMemoryStore() for the
// event store — the entire setup is in-memory (fileUsesMemoryEventStore exemption).
func TestC017_NoFindingWhenMemoryEventStoreInSameFile(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	eventStore := memory.NewMemoryStore()
	snapStore := memory.NewMemorySnapshotStore()
	_, _ = eventStore, snapStore
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings, err := correctness.NewC017Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings when memory event store in same file, got %d", len(findings))
	}
}

// C019: Multiple NewRepository for same type fires.
func TestC019_MultipleReposSameType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func a() {
	repo1 := decider.NewRepository[UserState](store1, bus1, d1)
	_ = repo1
}

func b() {
	repo2 := decider.NewRepository[UserState](store2, bus2, d2)
	_ = repo2
}
`,
	})

	findings, err := correctness.NewC019Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for duplicate repo, got %d", len(findings))
	}

	if string(findings[0].Rule) != "C019" {
		t.Errorf("expected C019, got %s", findings[0].Rule)
	}
}

// C019: No finding for single repo.
func TestC019_NoFindingForSingleRepo(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func a() {
	repo1 := decider.NewRepository[UserState](store1, bus1, d1)
	_ = repo1
}
`,
	})

	findings, err := correctness.NewC019Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

// C019: No finding for different types.
func TestC019_NoFindingForDifferentTypes(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func a() {
	repo1 := decider.NewRepository[UserState](store1, bus1, d1)
	repo2 := decider.NewRepository[OrderState](store2, bus2, d2)
	_, _ = repo1, repo2
}
`,
	})

	findings, err := correctness.NewC019Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for different types, got %d", len(findings))
	}
}

// C020: panic in SubscribeAll handler fires.
func TestC020_PanicInSubscribeAllHandler(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func setup() {
	bus.SubscribeAll(func(ctx context.Context, evt Event) error {
		id := evt.ID()
		if id == "" {
			panic("empty id")
		}
		return nil
	})
}
`,
	})

	findings, err := correctness.NewC020Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	if string(findings[0].Rule) != "C020" {
		t.Errorf("expected C020, got %s", findings[0].Rule)
	}
}

// C020: No panic in handler → no finding.
func TestC020_NoFindingForCleanHandler(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func setup() {
	bus.SubscribeAll(func(ctx context.Context, evt Event) error {
		return nil
	})
}
`,
	})

	findings, err := correctness.NewC020Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}
