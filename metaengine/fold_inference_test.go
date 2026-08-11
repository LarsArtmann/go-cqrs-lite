package metaengine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ── Test domain types ──

type inferUserCreated struct {
	ID     string
	Name   string
	Email  string
	Status string
}

type inferUserUpdated struct {
	ID     string
	Name   string
	Email  string
	Status string
}

type inferUserDeleted struct {
	ID string
}

type inferUserView struct {
	ID     string
	Name   string
	Email  string
	Status string
}

type getInferUser struct {
	ID string
}

type listInferUsers struct {
	Status string
}

type inferUserList struct {
	Items []inferUserView
}

// ── Nested struct test types ──

type inferAddress struct {
	City string
	Zip  string
}

type inferProfileCreated struct {
	ID      string
	Name    string
	Address inferAddress
}

type inferProfileView struct {
	ID   string
	Name string
	City string
	Zip  string
}

type getInferProfile struct {
	ID string
}

// ── Tests ──

func TestInfer_BasicCreateDelete(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getInferUser, inferUserView]("infer_basic",
		metaengine.Infer(inferUserCreated{}, inferUserDeleted{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "inferUserCreated", inferUserCreated{
		ID: "u1", Name: "Alice", Email: "alice@example.com", Status: "active",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "u1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "Alice" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice")
	}

	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@example.com")
	}

	if got.Status != "active" {
		t.Errorf("Status = %q, want %q", got.Status, "active")
	}

	if err := store.Apply(ctx, "inferUserDeleted", inferUserDeleted{ID: "u1"}); err != nil {
		t.Fatalf("Apply delete: %v", err)
	}

	_, err = metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "u1"},
	)
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Errorf("after delete: expected ErrNotFound, got %v", err)
	}
}

func TestInfer_FullCRUDLifecycle(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getInferUser, inferUserView]("infer_crud",
		metaengine.Infer(inferUserCreated{}, inferUserUpdated{}, inferUserDeleted{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create
	if err := store.Apply(ctx, "inferUserCreated", inferUserCreated{
		ID: "u1", Name: "Alice", Email: "alice@old.com", Status: "active",
	}); err != nil {
		t.Fatalf("Apply create: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "u1"},
	)
	if err != nil {
		t.Fatalf("after create: %v", err)
	}

	if got.Name != "Alice" || got.Email != "alice@old.com" {
		t.Fatalf("after create: got %+v", got)
	}

	// Update
	if err := store.Apply(ctx, "inferUserUpdated", inferUserUpdated{
		ID: "u1", Name: "Alice2", Email: "alice@new.com", Status: "suspended",
	}); err != nil {
		t.Fatalf("Apply update: %v", err)
	}

	got, err = metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "u1"},
	)
	if err != nil {
		t.Fatalf("after update: %v", err)
	}

	if got.Name != "Alice2" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice2")
	}

	if got.Email != "alice@new.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@new.com")
	}

	if got.Status != "suspended" {
		t.Errorf("Status = %q, want %q", got.Status, "suspended")
	}

	// Delete
	if err := store.Apply(ctx, "inferUserDeleted", inferUserDeleted{ID: "u1"}); err != nil {
		t.Fatalf("Apply delete: %v", err)
	}

	_, err = metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "u1"},
	)
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Errorf("after delete: expected ErrNotFound, got %v", err)
	}
}

func TestInfer_KeyFieldAutoDetected(t *testing.T) {
	t.Parallel()

	// The key field "ID" should be auto-detected from getInferUser.ID (string)
	// matching inferUserCreated.ID (string).
	q := metaengine.Query[getInferUser, inferUserView]("infer_key",
		metaengine.Infer(inferUserCreated{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "inferUserCreated", inferUserCreated{
		ID: "k1", Name: "Bob",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "k1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.ID != "k1" {
		t.Errorf("ID = %q, want %q", got.ID, "k1")
	}
}

func TestInfer_NestedStructFlattening(t *testing.T) {
	t.Parallel()

	// inferProfileCreated has a nested Address{City, Zip} struct.
	// inferProfileView has flat City and Zip fields.
	// The inference should flatten the nested struct and match fields by name.
	q := metaengine.Query[getInferProfile, inferProfileView]("infer_nested",
		metaengine.Infer(inferProfileCreated{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "inferProfileCreated", inferProfileCreated{
		ID:   "p1",
		Name: "Carol",
		Address: inferAddress{
			City: "Berlin",
			Zip:  "10115",
		},
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getInferProfile, inferProfileView](
		ctx, store, getInferProfile{ID: "p1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.City != "Berlin" {
		t.Errorf("City = %q, want %q", got.City, "Berlin")
	}

	if got.Zip != "10115" {
		t.Errorf("Zip = %q, want %q", got.Zip, "10115")
	}
}

func TestInfer_AutoFilterDetection(t *testing.T) {
	t.Parallel()

	// listInferUsers has a Status field that matches inferUserView.Status.
	// The planner should auto-detect this as a FilterOnField.
	// The result type is a collection (Items []inferUserView).
	q := metaengine.Query[listInferUsers, inferUserList]("infer_filter",
		metaengine.Infer(inferUserCreated{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "inferUserCreated", inferUserCreated{
		ID: "f1", Name: "Active User", Status: "active",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := store.Apply(ctx, "inferUserCreated", inferUserCreated{
		ID: "f2", Name: "Suspended User", Status: "suspended",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	results, err := metaengine.ExecuteTyped[listInferUsers, inferUserList](
		ctx, store, listInferUsers{Status: "active"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if len(results.Items) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results.Items))
	}

	if results.Items[0].Name != "Active User" {
		t.Errorf("Name = %q, want %q", results.Items[0].Name, "Active User")
	}
}

func TestInfer_PartialUpdate(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getInferUser, inferUserView]("infer_partial",
		metaengine.Infer(inferUserCreated{}, inferUserUpdated{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "inferUserCreated", inferUserCreated{
		ID: "u1", Name: "Alice", Email: "alice@example.com", Status: "active",
	}); err != nil {
		t.Fatalf("Apply create: %v", err)
	}

	// Partial update: only change Name, leave Email and Status as zero
	if err := store.Apply(ctx, "inferUserUpdated", inferUserUpdated{
		ID: "u1", Name: "AliceUpdated",
	}); err != nil {
		t.Fatalf("Apply update: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "u1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "AliceUpdated" {
		t.Errorf("Name = %q, want %q", got.Name, "AliceUpdated")
	}

	// Email should be preserved from the create event
	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q (should be preserved)", got.Email, "alice@example.com")
	}
}

func TestInfer_OnlyCreated(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getInferUser, inferUserView]("infer_only_created",
		metaengine.Infer(inferUserCreated{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "inferUserCreated", inferUserCreated{
		ID: "o1", Name: "Only Created",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "o1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "Only Created" {
		t.Errorf("Name = %q, want %q", got.Name, "Only Created")
	}
}

func TestInfer_ErrorNoCreated(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getInferUser, inferUserView]("infer_no_created",
		metaengine.Infer(inferUserDeleted{}),
	)

	eng := metaengine.NewMemoryEngine()
	_, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err == nil {
		t.Fatal("expected error for Infer with no Created sample")
	}
}

func TestInfer_ErrorUnrecognizedSuffix(t *testing.T) {
	t.Parallel()

	type WeirdEvent struct {
		ID string
	}

	q := metaengine.Query[getInferUser, inferUserView]("infer_weird",
		metaengine.Infer(inferUserCreated{}, WeirdEvent{}),
	)

	eng := metaengine.NewMemoryEngine()
	_, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err == nil {
		t.Fatal("expected error for unrecognized suffix")
	}
}

func TestInfer_ErrorInferPlusExplicitFolds(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for Infer + explicit folds")
		}
	}()

	metaengine.Query[getInferUser, inferUserView]("infer_mixed",
		metaengine.Infer(inferUserCreated{}),
		metaengine.AutoInsert[inferUserCreated, inferUserView]("ID"),
	)
}

func TestInfer_ErrorNoSamples(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for Infer with no samples")
		}
	}()

	metaengine.Infer()
}

func TestInfer_DryRun(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getInferUser, inferUserView]("infer_dryrun",
		metaengine.Infer(inferUserCreated{}, inferUserUpdated{}, inferUserDeleted{}),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng},
		q, metaengine.WithDryRun(),
	)
	if err != nil {
		t.Fatalf("Plan with dry run: %v", err)
	}
	defer store.Close()

	// In dry run, the store should still have the query registered
	// (asQueryMeta stores it even in dry run for plan inspection)
	// Verify the query exists by executing a query that should return ErrNotFound
	// (no events applied in dry run)
	ctx := context.Background()

	_, err = metaengine.ExecuteTyped[getInferUser, inferUserView](
		ctx, store, getInferUser{ID: "nonexistent"},
	)
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Errorf("expected ErrNotFound in dry run, got %v", err)
	}
}
