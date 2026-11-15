package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// RawQuery + Count mixing domain: the raw query consumes two events the
// Count projection does not — the exact shape that built a PARTIAL auto
// event decoder before raw decls contributed decoder entries.
type rqPetAdopted struct {
	ID        string
	Name      string
	AdoptedAt string
}

type rqPetRenamed struct {
	ID   string
	Name string
}

type rqPetView struct {
	ID        string
	Name      string
	AdoptedAt string
}

func TestRawQueryMixedWithProjectionSpecDecodesAllEvents(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Events: []event.Type{
			"rq.pet.adopted",
			"rq.pet.renamed",
		},
		Projections: []system.ProjectionDeclaration{
			// Declared mechanical folds over wire names; consumed only here.
			system.RawQuery(metaengine.Query[system.ScanInput, rqPetView]("rq_pets",
				metaengine.AutoInsertTyped[rqPetAdopted, rqPetView]("rq.pet.adopted", "ID"),
				metaengine.AutoUpdateTyped[rqPetRenamed, rqPetView]("rq.pet.renamed", "ID"),
			)),
			// A ProjectionSpec consuming a SUBSET of the raw query's events:
			// before the fix this made the auto decoder partial, so
			// rq.pet.renamed crashed the projection worker with
			// "no decoder registered".
			system.Count("rq-pet-counts").
				On("rq.pet.adopted", rqPetAdopted{}, +1, "adopted").
				Done(),
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	store := sys.MetaEngine()

	apply := func(wire string, payload any) {
		t.Helper()

		if err := store.Apply(ctx, wire, payload); err != nil {
			t.Fatalf("apply %s: %v", wire, err)
		}
	}

	apply("rq.pet.adopted", rqPetAdopted{ID: "p1", Name: "Rex", AdoptedAt: "2026-01-15"})
	apply("rq.pet.renamed", rqPetRenamed{ID: "p1", Name: "Rexy"})

	reader := metaengine.NewReader[rqPetView](store, "rq_pets")

	deadline := time.Now().Add(5 * time.Second)

	for {
		v, found, err := reader.Get(ctx, "p1")
		if err != nil {
			t.Fatalf("lookup: %v (poisoned: %v)", err, store.IsPoisoned("rq_pets"))
		}

		if found && v.Name == "Rexy" && v.AdoptedAt == "2026-01-15" {
			break
		}

		if time.Now().After(deadline) {
			perr := store.IsPoisoned("rq_pets")

			t.Fatalf("rename never folded: found=%v view=%+v poisoned=%v", found, v, perr)
		}

		time.Sleep(20 * time.Millisecond)
	}

	counts, err := system.GetCount(ctx, sys, "rq-pet-counts")
	if err != nil {
		t.Fatalf("count: %v", err)
	}

	if counts["adopted"] != 1 {
		t.Fatalf("counts = %+v, want adopted=1", counts)
	}
}
