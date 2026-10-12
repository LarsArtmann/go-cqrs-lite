package systemscenario_test

// F75: the harness must behave identically when the decider repository is
// snapshot-configured — the write side crosses the strategy and the read
// side rebuilds through snapshot+replay, and neither may distort the journal
// or the decisions drawn from state.

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// snapshotTaskDomain is taskDomain with a snapshot strategy attached to the
// Task decider — the same handlers, one registration option different.
func snapshotTaskDomain() system.DomainConfig {
	base := taskDomain()
	base.Commands = func(sys *system.System) {
		strategy, err := snapshot.EveryNEvents(2)
		if err != nil {
			panic(err)
		}

		if err := system.RegisterDecider(sys, "Task", taskDecider, system.WithSnapshotStrategy(strategy)); err != nil {
			panic(err)
		}

		registerTaskCommandHandlers(sys)
	}

	return base
}

// snapshotDeployment adds a dedicated snapshots instance to the memory
// preset: the memory engine does NOT implement SnapshotBackend, so the
// snapshot store lives on a file-backed SQLite engine (the same wiring
// systemtest's dedicated-snapshots test pins).
func snapshotDeployment(t testing.TB) system.DeploymentConfig {
	deploy := systemscenario.Memory()
	deploy.Engines["snaps"] = system.EngineConfig{
		Driver:  "sqlite",
		DSN:     filepath.Join(t.TempDir(), "snaps.db"),
		Pragmas: []string{"journal_mode=wal"},
	}
	deploy.Instances = append(deploy.Instances, system.InstanceConfig{
		Role:   system.RoleSnapshots,
		Engine: "snaps",
	})

	return deploy
}

func newSnapshotScenario(t *testing.T) (*systemscenario.Scenario, id.StreamRef, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(t, ctx, snapshotTaskDomain(), snapshotDeployment(t))

	return sc, ref, ctx
}

func TestScenario_SnapshotConfiguredDecider(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newSnapshotScenario(t)

	// Seed by intent: create (v1) + rename (v2) crosses EveryNEvents(2), so
	// the write side must have persisted a v2 snapshot.
	sc.Given().
		Command(newTaskCmd("task.create", ref.ID)).
		Command(newTaskCmd("task.rename", ref.ID))

	snap, err := sc.System().SnapshotStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("snapshot after 2 events: %v", err)
	}

	if snap.Version != 2 {
		t.Fatalf("snapshot version: want 2, got %d", snap.Version)
	}

	// The read side rebuilds state through snapshot+replay: completion must
	// see the renamed title — decisions identical to the non-snapshot path.
	sc.When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated").
		ThenEvents(func(events []event.Event) {
			systemscenario.ThenPayload(sc.Phase(), 0, TaskUpdated{
				ID:     ref.ID.String(),
				Title:  "renamed",
				Status: "completed",
			})
		})
}

func TestScenario_SnapshotDeploymentJournalEquivalence(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newTaskScenario(t)
	snapSc, snapRef, _ := newSnapshotScenario(t)

	// Different minted IDs per system are fine — the assertion normalizes
	// streams by first-appearance ordinal, which is the point under test.
	seed := func(sc *systemscenario.Scenario, ref id.StreamRef) {
		sc.Given().
			Command(newTaskCmd("task.create", ref.ID)).
			Command(newTaskCmd("task.rename", ref.ID)).
			Command(newTaskCmd("task.complete", ref.ID))
	}

	seed(sc, ref)
	seed(snapSc, snapRef)

	// A final identical act on both satisfies the vacuous-pass guard and
	// proves the snapshot deployment keeps deciding from the same state.
	sc.When(newTaskCmd("task.rename", ref.ID)).Then("task.updated")
	snapSc.When(newTaskCmd("task.rename", snapRef.ID)).Then("task.updated")

	systemscenario.AssertJournalEquivalence(t, ctx, sc.System(), snapSc.System())
}
