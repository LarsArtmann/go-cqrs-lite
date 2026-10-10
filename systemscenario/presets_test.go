package systemscenario_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// bootPresetScenario boots the fixture domain on a preset deployment and
// runs one command; assertions ride the returned WhenPhase so preset tests
// assert through the real harness flow.
func bootPresetScenario(
	t *testing.T,
	deploy system.DeploymentConfig,
) (*systemscenario.Scenario, *systemscenario.WhenPhase, id.StreamRef) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(t, ctx, taskDomain(), deploy)

	phase := sc.Given(
		sc.Event(
			"task.created",
			ref,
			TaskCreated{ID: ref.ID.String(), Title: "preset", Status: "pending"},
		),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated")

	return sc, phase, ref
}

// TestPreset_MemoryBootsScenario pins the Memory() one-liner: the whole
// Given/When/Then flow runs against memory engines with zero deployment
// boilerplate.
func TestPreset_MemoryBootsScenario(t *testing.T) {
	t.Parallel()

	bootPresetScenario(t, systemscenario.Memory())
}

// TestPreset_SQLiteBootsScenario pins the SQLite(t) one-liner: the same
// flow runs against a fresh file-backed SQLite database under t.TempDir().
func TestPreset_SQLiteBootsScenario(t *testing.T) {
	t.Parallel()

	bootPresetScenario(t, systemscenario.SQLite(t))
}

// TestPreset_SQLiteThenQuerySeesProjection extends the SQLite boot with an
// awaited read-model assertion — the preset must wire projections, not just
// the journal.
func TestPreset_SQLiteThenQuerySeesProjection(t *testing.T) {
	t.Parallel()

	sc, phase, ref := bootPresetScenario(t, systemscenario.SQLite(t))

	phase.ThenQueryFunc(func() (any, error) {
		return taskViewQuery(sc, context.Background(), ref.ID.String())()
	}, func(got any) error {
		view, ok := got.(TaskView)
		if !ok {
			return fmt.Errorf("want TaskView, got %T", got)
		}
		if view.Status != "completed" {
			return fmt.Errorf("want status completed, got %s", view.Status)
		}
		return nil
	})
}

// TestPreset_MemoryResolvesTimersEngine pins that Memory() declares the
// dedicated timers engine, so TimeAdvances scenarios work out of the box
// (the full firing flow is covered by timer_test.go via timerDeployment,
// which mirrors this layout). Adopt keeps this a plain wiring check — no
// harness chain, so the vacuous-pass guard stays silent.
func TestPreset_MemoryResolvesTimersEngine(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, taskDomain(), systemscenario.Memory())
	if err != nil {
		t.Fatalf("system.New on Memory(): %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	if sys.TimerEngine() == nil {
		t.Fatal("Memory() preset must resolve a timers engine")
	}
}
