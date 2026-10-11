package systemtest_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_LifecycleStress_Sqlite is the real-engine lifecycle/shutdown
// stress leg (Feedback #6 (b) / M17.3): repeated construct → start →
// concurrent dispatch → drain → racing-close cycles against the real sqlite
// driver. The Count projection is the double-apply sentinel — after the drain
// the counter must equal the number of dispatched events exactly; any replay
// or double-delivery during shutdown races shows up as over-counting.
func TestSystem_LifecycleStress_Sqlite(t *testing.T) {
	t.Parallel()

	rounds := 6
	if testing.Short() {
		rounds = 2
	}

	const workers = 8

	for round := range rounds {
		t.Run(fmt.Sprintf("round-%d", round), func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			sys, err := system.New(ctx, stressDomain(), system.DeploymentConfig{
				Engines: map[string]system.EngineConfig{
					"primary": {
						Driver: "sqlite",
						DSN:    "file:" + filepath.Join(t.TempDir(), "stress.db"),
					},
				},
				Instances: []system.InstanceConfig{
					{Role: system.RoleSourceOfTruth, Engine: "primary"},
					{Role: system.RoleProjections, Engine: "primary"},
				},
			})
			if err != nil {
				t.Fatalf("round %d: system.New: %v", round, err)
			}

			if err := sys.Start(ctx); err != nil {
				t.Fatalf("round %d: Start: %v", round, err)
			}

			if err := dispatchConcurrentCreates(ctx, sys, workers); err != nil {
				t.Fatalf("round %d: %v", round, err)
			}

			waitForProcessed(t, sys, workers)

			counts, err := system.GetCount(ctx, sys, "stress_counts")
			if err != nil {
				t.Fatalf("round %d: GetCount: %v", round, err)
			}

			if got := counts["created"]; got != workers {
				t.Fatalf("round %d: created count = %d, want exactly %d "+
					"(under-count = lost event, over-count = double-apply)",
					round, got, workers)
			}

			if err := raceClosers(ctx, sys); err != nil {
				t.Fatalf("round %d: %v", round, err)
			}
		})
	}
}

// stressDomain is the stress fixture: a Task decider whose create command
// emits stress.created, projected into a Count (the double-apply sentinel).
func stressDomain() system.DomainConfig {
	return system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "stress.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if state.Exists {
								return nil, errors.New("already exists")
							}

							return []event.Event{mustEvent(event.New(
								"stress.created",
								cmd.StreamID(),
								"Task",
								ver+1,
								TaskCreated{Title: "stress", At: time.Now()},
							))}, nil
						})
				})
		},
		Projections: []system.ProjectionDeclaration{
			system.Count("stress_counts").
				On("stress.created", TaskCreated{}, +1, "created").
				Done(),
		},
	}
}

// dispatchConcurrentCreates fans out n create commands on distinct streams.
func dispatchConcurrentCreates(ctx context.Context, sys *system.System, n int) error {
	var wg sync.WaitGroup

	errCh := make(chan error, n)

	for range n {

		wg.Go(func() {

			errCh <- sys.CommandDispatcher().
				Dispatch(ctx, newCmd("stress.create", id.NewStreamID()))
		})
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return fmt.Errorf("dispatch: %w", err)
		}
	}

	return nil
}

// waitForProcessed blocks until the projection host processed at least n
// events with zero errors, or the load-scaled deadline lapses.
func waitForProcessed(t *testing.T, sys *system.System, n int) {
	t.Helper()

	deadline := loadScaledDeadline(20 * time.Second)

	for time.Now().Before(deadline) {
		var processed int64
		var errs int64

		for _, s := range sys.ProjectionHost().Status() {
			processed += s.Processed
			errs += s.Errors
		}

		if processed >= int64(n) && errs == 0 {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// raceClosers runs GracefulClose and Close from different goroutines — both
// must finish without error; double-close idempotency under a real race.
func raceClosers(ctx context.Context, sys *system.System) error {
	var wg sync.WaitGroup

	errs := make(chan error, 2)
	wg.Add(2)

	go func() {
		defer wg.Done()

		errs <- sys.GracefulClose(ctx)
	}()

	go func() {
		defer wg.Done()

		errs <- sys.Close()
	}()

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return fmt.Errorf("racing close: %w", err)
		}
	}

	return nil
}
