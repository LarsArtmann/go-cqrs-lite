package metaengine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Typed-fold domain: wire names deliberately differ from Go struct names and
// carry no Created/Updated/Deleted suffix — the fold kind is declared, not
// inferred from the payload name.
type typedPetAdopted struct {
	ID        string
	Name      string
	AdoptedAt string
}

// Two string fields prove the key extractor matches by keyField NAME (the
// auto cores pre-build extractors; deriveKeys' type-matching never runs).
type typedPetRenamed struct {
	ID   string
	Name string
}

type typedPetReleased struct {
	ID string
}

type typedPetView struct {
	ID        string
	Name      string
	AdoptedAt string
}

type getTypedPet struct {
	ID string
}

func planTypedPets(t *testing.T, q metaengine.QueryDecl[getTypedPet, typedPetView]) *metaengine.Store {
	t.Helper()

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store
}

func TestAutoInsertTypedBindsWireName(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getTypedPet, typedPetView](
		"typed_insert",
		metaengine.AutoInsertTyped[typedPetAdopted, typedPetView]("pet.adopted", "ID"),
	)

	if got := q.QueryFolds()[0].EventType(); got != "pet.adopted" {
		t.Fatalf("fold event type = %q, want wire name %q", got, "pet.adopted")
	}
}

func TestAutoInsertTypedRoundTrip(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getTypedPet, typedPetView](
		"typed_insert_rt",
		metaengine.AutoInsertTyped[typedPetAdopted, typedPetView]("pet.adopted", "ID"),
	)

	store := planTypedPets(t, q)
	ctx := context.Background()

	if err := store.Apply(ctx, "pet.adopted", typedPetAdopted{
		ID: "p1", Name: "Rex", AdoptedAt: "2026-11-15",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getTypedPet, typedPetView](
		ctx, store, getTypedPet{ID: "p1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "Rex" || got.AdoptedAt != "2026-11-15" {
		t.Fatalf("view = %+v, want field-copied adopt", got)
	}
}

func TestAutoUpdateTypedTwoStringFieldsNoAmbiguity(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getTypedPet, typedPetView](
		"typed_update",
		metaengine.AutoInsertTyped[typedPetAdopted, typedPetView]("pet.adopted", "ID"),
		metaengine.AutoUpdateTyped[typedPetRenamed, typedPetView]("pet.renamed", "ID"),
	)

	store := planTypedPets(t, q)
	ctx := context.Background()

	if err := store.Apply(ctx, "pet.adopted", typedPetAdopted{
		ID: "p1", Name: "Rex", AdoptedAt: "2026-11-15",
	}); err != nil {
		t.Fatalf("Apply adopt: %v", err)
	}

	if err := store.Apply(ctx, "pet.renamed", typedPetRenamed{ID: "p1", Name: "Rexy"}); err != nil {
		t.Fatalf("Apply rename: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getTypedPet, typedPetView](
		ctx, store, getTypedPet{ID: "p1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "Rexy" || got.AdoptedAt != "2026-11-15" {
		t.Fatalf("view = %+v, want renamed with adoptedAt preserved", got)
	}
}

func TestAutoDeleteTypedRemovesRow(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getTypedPet, typedPetView](
		"typed_delete",
		metaengine.AutoInsertTyped[typedPetAdopted, typedPetView]("pet.adopted", "ID"),
		metaengine.AutoDeleteTyped[typedPetReleased]("pet.released", "ID"),
	)

	store := planTypedPets(t, q)
	ctx := context.Background()

	if err := store.Apply(ctx, "pet.adopted", typedPetAdopted{ID: "p1", Name: "Rex"}); err != nil {
		t.Fatalf("Apply adopt: %v", err)
	}

	if err := store.Apply(ctx, "pet.released", typedPetReleased{ID: "p1"}); err != nil {
		t.Fatalf("Apply release: %v", err)
	}

	_, err := metaengine.ExecuteTyped[getTypedPet, typedPetView](
		ctx, store, getTypedPet{ID: "p1"},
	)
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Fatalf("after pet.released: err = %v, want ErrNotFound", err)
	}
}

func TestAutoCRUDTypedDeclaresAllWireNames(t *testing.T) {
	t.Parallel()

	folds := metaengine.AutoCRUDTyped[
		typedPetAdopted, typedPetRenamed, typedPetReleased, typedPetView,
	]("pet.adopted", "pet.renamed", "pet.released", "ID")

	want := []string{"pet.adopted", "pet.renamed", "pet.released"}

	if len(folds) != len(want) {
		t.Fatalf("got %d folds, want %d", len(folds), len(want))
	}

	for i, wire := range want {
		if got := folds[i].EventType(); got != wire {
			t.Fatalf("fold[%d] event type = %q, want %q", i, got, wire)
		}
	}
}

// TestAutoCRUDTypedScanRoundTrip proves the declared-fold path end to end
// through a scan query: insert, partial-merge update, delete, and a runtime
// filter — all under wire names that never match the payload type names.
func TestAutoCRUDTypedScanRoundTrip(t *testing.T) {
	t.Parallel()

	type scanPets struct{}

	folds := metaengine.AutoCRUDTyped[
		typedPetAdopted, typedPetRenamed, typedPetReleased, typedPetView,
	]("pet.adopted", "pet.renamed", "pet.released", "ID")

	args := make([]any, 0, len(folds)+2)
	for _, f := range folds {
		args = append(args, f)
	}
	args = append(args,
		metaengine.FilterOnField[typedPetView]("name", metaengine.FilterEq),
		metaengine.SortOnField[typedPetView]("adoptedAt", false),
	)

	q := metaengine.Query[scanPets, typedPetView]("typed_pets_scan", args...)

	eng := metaengine.NewMemoryEngine()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = store.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apply := func(wire string, payload any) {
		t.Helper()

		if err := store.Apply(ctx, wire, payload); err != nil {
			t.Fatalf("Apply %s: %v", wire, err)
		}
	}

	apply("pet.adopted", typedPetAdopted{ID: "p1", Name: "Rex", AdoptedAt: "2026-01-15"})
	apply("pet.adopted", typedPetAdopted{ID: "p2", Name: "Fido", AdoptedAt: "2026-01-16"})
	apply("pet.renamed", typedPetRenamed{ID: "p2", Name: "Fido Jr."})
	apply("pet.released", typedPetReleased{ID: "p1"})

	reader := metaengine.NewReader[typedPetView](store, "typed_pets_scan")

	deadline := time.Now().Add(5 * time.Second)

	for {
		pets, err := reader.Scan(ctx, metaengine.WithFilter("name", metaengine.FilterEq, "Fido Jr."))
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}

		if len(pets) == 1 && pets[0].ID == "p2" && pets[0].AdoptedAt == "2026-01-16" {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("filtered scan never settled; last = %+v", pets)
		}

		//go-sleep:accept poll-interval spacing inside the deadline-bounded retry loop above (eventual consistency settle)
		time.Sleep(20 * time.Millisecond)
	}
}
