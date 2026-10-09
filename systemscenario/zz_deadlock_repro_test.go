package systemscenario_test

// TEMPORARY repro for the deriver-bus deadlock evidence pack (adoption-wave
// T01). Deleted after stack capture; NOT a regression test — after ADR-0154
// decision (b) lands, a sync deriver still deadlocks by design until the
// watermill reentrancy loud-fail (T26) converts the hang into an error.

import (
	"bytes"
	"context"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/deriver/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// syncSagaDomain wires the archiver saga WITHOUT the goroutine workaround:
// the deriver returns the derived command synchronously, so AsHandler
// dispatches it inside the bus handler.
func syncSagaDomain(t *testing.T) system.DomainConfig {
	t.Helper()

	base := taskDomain()
	baseCommands := base.Commands

	base.Commands = func(sys *system.System) {
		baseCommands(sys)
		registerArchive(sys)

		archiver := deriver.Deriver(
			func(ctx context.Context, evt event.Event) ([]command.Command, error) {
				return []command.Command{newTaskCmd("task.archive", evt.StreamID())}, nil
			},
		)
		if err := sys.Bus().
			Subscribe("task.updated", archiver.AsHandler(sys.CommandDispatcher())); err != nil {
			t.Fatal(err)
		}
	}

	return base
}

func TestZZDeadlockReproSyncDeriver(t *testing.T) {
	ctx := context.Background()

	sys, err := system.New(ctx, syncSagaDomain(t), memoryDeployment())
	if err != nil {
		t.Fatal(err)
	}
	if err := sys.Start(ctx); err != nil {
		t.Fatal(err)
	}
	// Deliberately NO cleanup: the system is expected to deadlock and Close
	// would hang on the stuck publish drain.

	ref := id.NewStreamRef("Task", id.NewStreamID())

	done := make(chan error, 1)
	go func() {
		if err := sys.CommandDispatcher().Dispatch(ctx, newTaskCmd("task.create", ref.ID)); err != nil {
			done <- err
			return
		}
		done <- sys.CommandDispatcher().Dispatch(ctx, newTaskCmd("task.complete", ref.ID))
	}()

	select {
	case err := <-done:
		t.Fatalf("dispatch returned (err=%v) — deadlock NOT reproduced", err)
	case <-time.After(3 * time.Second):
	}

	var buf bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&buf, 2); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/deriver-deadlock-stacks.txt", buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
