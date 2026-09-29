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

			domain := system.DomainConfig{
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

			deployment := system.DeploymentConfig{
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
			}

			sys, err := system.New(ctx, domain, deployment)
			if err != nil {
				t.Fatalf("round %d: system.New: %v", round, err)
			}

			if err := sys.Start(ctx); err != nil {
				t.Fatalf("round %d: Start: %v", round, err)
			}

			var wg sync.WaitGroup

			errCh := make(chan error, workers)

			for range workers {
				wg.Add(1)

				go func() {
					defer wg.Done()

					errCh <- sys.CommandDispatcher().
						Dispatch(ctx, newCmd("stress.create", id.NewStreamID()))
				}()
			}

			wg.Wait()
			close(errCh)

			for err := range errCh {
				if err != nil {
					t.Fatalf("round %d: dispatch: %v", round, err)
				}
			}

			deadline := loadScaledDeadline(20 * time.Second)

			for time.Now().Before(deadline) {
				var processed int64
				errs := 0

				for _, s := range sys.ProjectionHost().Status() {
					processed += s.Processed
					errs += s.Errors
				}

				if processed >= workers && errs == 0 {
					break
				}

				time.Sleep(50 * time.Millisecond)
			}

			counts, err := system.GetCount(ctx, sys, "stress_counts")
			if err != nil {
				t.Fatalf("round %d: GetCount: %v", round, err)
			}

			if got := counts["created"]; got != workers {
				t.Fatalf("round %d: created count = %d, want exactly %d "+
					"(under-count = lost event, over-count = double-apply)",
					round, got, workers)
			}

			// Racing shutdown: GracefulClose and Close from different
			// goroutines must both finish without error; double-close is
			// pinned idempotent elsewhere, this pins it under a real race.
			var closeWG sync.WaitGroup

			closeErrs := make(chan error, 2)
			closeWG.Add(2)

			go func() {
				defer closeWG.Done()

				closeErrs <- sys.GracefulClose(ctx)
			}()

			go func() {
				defer closeWG.Done()

				closeErrs <- sys.Close()
			}()

			closeWG.Wait()
			close(closeErrs)

			for err := range closeErrs {
				if err != nil {
					t.Fatalf("round %d: racing close: %v", round, err)
				}
			}
		})
	}
}
