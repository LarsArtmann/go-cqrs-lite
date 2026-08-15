package metaengine

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// renamedEngine gives a memory engine a distinct Profile().Name so multiple
// memory engines can coexist in one store. Embedding the concrete type keeps
// every backend interface (MapBackend, ScanBackend, ...) promoted.
type renamedEngine struct {
	*memoryEngine

	name string
}

func (r *renamedEngine) Profile() EngineProfile {
	p := r.memoryEngine.Profile()
	p.Name = r.name

	return p
}

func renamed(name string) Engine {
	return &renamedEngine{memoryEngine: NewMemoryEngine().(*memoryEngine), name: name}
}

func roleTestStore(t *testing.T) (*Store, Engine) {
	t.Helper()

	primary := NewMemoryEngine()
	store, err := Plan([]Engine{primary}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = store.Close() })

	return store, primary
}

type (
	roleItemCreated struct{ ID, Name string }
	roleItem        struct{ Name string }
	roleFindItem    struct{ ID string }
)

func roleItemQuery() any {
	return Query[roleFindItem, roleItem](
		"role_items",
		OnRecord(roleItemCreated{}, func(_ record.Record, e roleItemCreated) (string, roleItem) {
			return e.ID, roleItem{Name: e.Name}
		}),
	)
}

func TestEngineRole_DefaultAndLookup(t *testing.T) {
	t.Parallel()

	store, _ := roleTestStore(t)

	role, ok := store.EngineRole("memory")
	if !ok || role != RoleActive {
		t.Fatalf("plan-time engine should default to Active, got %q ok=%v", role, ok)
	}

	if _, ok := store.EngineRole("nope"); ok {
		t.Fatal("unknown engine should not be found")
	}
}

func TestAddEngine_InvalidRoleRejected(t *testing.T) {
	t.Parallel()

	store, _ := roleTestStore(t)

	err := store.AddEngine(context.Background(), renamed("bad"), WithEngineRole("Wat"))
	if err == nil {
		t.Fatal("invalid role must be rejected")
	}
}

// TestShadowEngine_NeverRouted proves invariant I1: a Backup engine is excluded
// from routing — every query stays on the primary despite identical profiles.
func TestShadowEngine_NeverRouted(t *testing.T) {
	t.Parallel()

	store, _ := roleTestStore(t)

	if err := store.AddEngine(
		context.Background(),
		renamed("shadow"),
		WithEngineRole(RoleBackup),
	); err != nil {
		t.Fatal(err)
	}

	for _, qa := range store.Plan().Queries {
		if qa.EngineName == "shadow" {
			t.Fatalf("query %q routed to shadow engine — violates invariant I1", qa.QueryName)
		}
	}

	if role, _ := store.EngineRole("shadow"); role != RoleBackup {
		t.Fatalf("role should be Backup, got %q", role)
	}
}

func TestPromoteEngine_RejectsNonShadow(t *testing.T) {
	t.Parallel()

	store, _ := roleTestStore(t)

	if err := store.PromoteEngine(context.Background(), "memory"); err == nil {
		t.Fatal("promoting an Active engine must fail")
	}

	if err := store.PromoteEngine(context.Background(), "ghost"); err == nil {
		t.Fatal("promoting an unknown engine must fail")
	}
}

func TestProjectionRoleHelpers(t *testing.T) {
	t.Parallel()

	if !RoleActive.routable() || !RoleDualUse.routable() {
		t.Fatal("Active/DualUse must be routable")
	}

	if RoleMigration.routable() || RoleBackup.routable() {
		t.Fatal("Migration/Backup must not be routable")
	}

	if !RoleMigration.IsShadow() || !RoleBackup.IsShadow() {
		t.Fatal("Migration/Backup must be shadows")
	}

	if RoleActive.IsShadow() || RoleDualUse.IsShadow() {
		t.Fatal("Active/DualUse must not be shadows")
	}

	if ProjectionRole("bogus").Valid() {
		t.Fatal("bogus role must be invalid")
	}
}

// TestDualUseEngine_IsRoutable proves DualUse engines participate in routing.
func TestDualUseEngine_IsRoutable(t *testing.T) {
	t.Parallel()

	store, _ := roleTestStore(t)

	if err := store.AddEngine(
		context.Background(),
		renamed("dual"),
		WithEngineRole(RoleDualUse),
	); err != nil {
		t.Fatal(err)
	}

	if _, ok := store.EngineRole("dual"); !ok {
		t.Fatal("dual engine not registered")
	}

	if role, _ := store.EngineRole("dual"); role != RoleDualUse {
		t.Fatalf("role should be DualUse, got %q", role)
	}
}
