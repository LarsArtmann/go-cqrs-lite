package systemscenario_test

// Chaos-leg self-tests: the DelayedDriver seam injects journal latency and
// proves the harness invariants (ordering, fold-vs-read-model, optimistic
// concurrency) hold when the journal is slow.

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// delayedMemoryDeployment is the Memory preset with its primary engine's
// journal wrapped in latency chaos; timers stay on the undelayed dedicated
// engine.
func delayedMemoryDeployment(t *testing.T, delay time.Duration) system.DeploymentConfig {
	t.Helper()

	deploy := systemscenario.Memory()
	deploy.Engines["primary"] = system.EngineConfig{
		Driver: systemscenario.DelayedDriver(t, "memory", delay),
	}

	return deploy
}

// newDelayedScenario boots the fixture domain on the delayed deployment.
func newDelayedScenario(t *testing.T) (*systemscenario.Scenario, id.StreamRef, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(
		t,
		ctx,
		taskDomain(),
		delayedMemoryDeployment(t, 2*time.Millisecond),
	)

	return sc, ref, ctx
}

// TestChaosDelayedJournal_ReadModelMatchesUnderLatency boots a full scenario
// through the delayed journal. Boot alone proves capability forwarding:
// system.New's fail-closed gate rejects engines that expose neither
// AtomicAppender nor Transactional, and the delayed wrapper passes it only
// through explicit forwarding.
func TestChaosDelayedJournal_ReadModelMatchesUnderLatency(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newDelayedScenario(t)

	sc.Given(
		sc.Event(
			"task.created",
			ref,
			TaskCreated{ID: ref.ID.String(), Title: "ship it", Status: "pending"},
		),
	).When(newTaskCmd("task.rename", ref.ID)).
		Command(newTaskCmd("task.complete", ref.ID)).
		ThenQuery(
			taskViewQuery(sc, ctx, ref.ID.String()),
			TaskView{ID: ref.ID.String(), Title: "renamed", Status: "completed"},
		)
}

// TestChaosDelayedJournal_OptimisticConcurrencyHolds races two concurrent
// saves at the same expected version through the delayed wrapper's
// forwarded AtomicAppender path: exactly one may win, the other must fail
// with a version conflict. Latency in front of the check-then-append must
// not widen into a lost-update window.
func TestChaosDelayedJournal_OptimisticConcurrencyHolds(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newDelayedScenario(t)
	sys := sc.System()

	sc.Given(
		sc.Event(
			"task.created",
			ref,
			TaskCreated{ID: ref.ID.String(), Title: "ship it", Status: "pending"},
		),
	).When(newTaskCmd("task.rename", ref.ID)).
		ThenSuccess()

	var (
		wg              sync.WaitGroup
		successfulSaves atomic.Int32
	)

	for range 2 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			evt := taskEvent(
				"task.updated",
				ref.ID,
				event.Version(3),
				TaskUpdated{ID: ref.ID.String(), Title: "racy", Status: "pending"},
			)

			// The stream is at version 2 (created + rename); both racers
			// claim version 2, so the atomic append must serialize them.
			err := sys.EventStore().Save(ctx, ref, []event.Event{evt}, event.Version(2))
			if err == nil {
				successfulSaves.Add(1)
			}
		}()
	}

	wg.Wait()

	if got := successfulSaves.Load(); got != 1 {
		t.Fatalf(
			"optimistic concurrency under latency: expected exactly 1 of 2 concurrent saves to win, got %d",
			got,
		)
	}
}
