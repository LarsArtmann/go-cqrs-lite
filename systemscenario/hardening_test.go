package systemscenario_test

// Hardening pack tests (T16/T17 of the 2026-10-09 BDD harness adoption
// wave): await-mode ThenCommandsSatisfy, timeout last-error surfacing,
// ThenQueryEventuallyFails, the ThenNoEvents quiet window, and the command
// capture filter.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// sentinelFailed unwinds a failTB like runtime.Goexit unwinds a real T after
// Fatal: later statements in the scenario chain must not run.
type sentinelFailed struct{}

// failTB records Fatal*/Error* calls instead of failing a real test, so
// failure-path tests can assert the harness's diagnostics verbatim. The
// embedded nil testing.TB satisfies the interface; every method the harness
// calls is implemented below.
type failTB struct {
	testing.TB

	mu       sync.Mutex
	messages []string
	cleanups []func()
}

func (f *failTB) Helper() {}

func (f *failTB) Cleanup(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cleanups = append(f.cleanups, fn)
}

func (f *failTB) runCleanups() {
	f.mu.Lock()
	cleanups := f.cleanups
	f.mu.Unlock()

	for _, fn := range cleanups {
		fn()
	}
}

func (f *failTB) record(message string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.messages = append(f.messages, message)
}

func (f *failTB) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.messages...)
}

func (f *failTB) Fatalf(format string, args ...any) {
	f.record(fmt.Sprintf(format, args...))
	panic(sentinelFailed{})
}

func (f *failTB) Fatal(args ...any) {
	f.Fatalf("%s", fmt.Sprint(args...))
}

func (f *failTB) Errorf(format string, args ...any) {
	f.record(fmt.Sprintf(format, args...))
}

// runExpectingFailure boots a scenario over failTB and returns the recorded
// diagnostics. Registered cleanups run exactly once (system shutdown) no
// matter how run unwinds — the sentinel panic unwinds like Goexit, so the
// named return is set in the defer.
func runExpectingFailure(t *testing.T, run func(tb testing.TB)) (messages []string) {
	t.Helper()

	ftb := &failTB{}

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(sentinelFailed); !ok {
				panic(r)
			}
		}

		ftb.runCleanups()
		messages = ftb.snapshot()
	}()

	run(ftb)

	return ftb.snapshot()
}

// ── ThenCommandsSatisfyAwait (F16.1) ──

// TestThenCommandsSatisfyAwait_PassesOnDerivedChain: the archiver saga
// dispatches task.archive asynchronously, so awaiting inspection sees it
// once the bus delivery lands.
func TestThenCommandsSatisfyAwait_PassesOnDerivedChain(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newSagaScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Await().
		ThenCommandsSatisfyAwait(func(cmds []command.Command) error {
			for _, cmd := range cmds {
				if cmd.Type() == "task.archive" && cmd.StreamID() == ref.ID {
					return nil
				}
			}

			return fmt.Errorf(
				"task.archive on %s not dispatched yet (%d captured)",
				ref.ID,
				len(cmds),
			)
		})
}

// TestThenCommandsSatisfyAwait_TimeoutSurfacesLastInspectError: the timeout
// message must carry the LAST inspect error, not a generic timeout.
func TestThenCommandsSatisfyAwait_TimeoutSurfacesLastInspectError(t *testing.T) {
	t.Parallel()

	messages := runExpectingFailure(t, func(tb testing.TB) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ref := id.NewStreamRef("Task", id.NewStreamID())
		sc := systemscenario.System(tb, ctx, taskDomain(), memoryDeployment(),
			systemscenario.WithAwaitTimeout(60*time.Millisecond))

		sc.Given().Command(newTaskCmd("task.create", ref.ID)).
			When(newTaskCmd("task.rename", ref.ID)).
			Await().
			ThenCommandsSatisfyAwait(func(cmds []command.Command) error {
				return fmt.Errorf("no derived command among %d captured", len(cmds))
			})
	})

	if len(messages) == 0 {
		t.Fatal("expected a timeout failure, got none")
	}

	last := messages[len(messages)-1]
	// The Given-phase create predates the act baseline; only the When-act
	// rename is captured after it.
	if !contains(last, "no derived command among 1 captured") {
		t.Fatalf("timeout message must surface the last inspect error, got: %s", last)
	}
}

// ── Timeout last-error surfacing (F16.2) ──

// TestThenQueryFunc_TimeoutReportsLastQueryError: a query that errored
// before the data settled must not be hidden by a later check mismatch —
// the timeout detail reports both channels.
func TestThenQueryFunc_TimeoutReportsLastQueryError(t *testing.T) {
	t.Parallel()

	messages := runExpectingFailure(t, func(tb testing.TB) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ref := id.NewStreamRef("Task", id.NewStreamID())
		sc := systemscenario.System(tb, ctx, taskDomain(), memoryDeployment(),
			systemscenario.WithAwaitTimeout(60*time.Millisecond))

		calls := 0

		sc.Given().Command(newTaskCmd("task.create", ref.ID)).
			When(newTaskCmd("task.rename", ref.ID)).
			ThenQueryFunc(
				func() (any, error) {
					calls++
					if calls == 1 {
						return nil, errors.New("view not projected yet")
					}

					return "stale shape", nil
				},
				func(got any) error {
					return fmt.Errorf("result mismatch: want fresh shape, got %v", got)
				},
			)
	})

	if len(messages) == 0 {
		t.Fatal("expected a timeout failure, got none")
	}

	last := messages[len(messages)-1]
	if !contains(last, "result mismatch") ||
		!contains(last, "last query error: view not projected yet") {
		t.Fatalf("timeout detail must report check message AND last query error, got: %s", last)
	}
}

// ── ThenQueryEventuallyFails (the awaitNotFound blind spot, F16.2) ──

// TestThenQueryEventuallyFails_PassesWhenLookupStartsFailing: the failing
// lookup itself is the awaited condition — first-class awaitNotFound.
func TestThenQueryEventuallyFails_PassesWhenLookupStartsFailing(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	errGone := errorfamily.NewRejection("task.gone", "task gone")

	calls := 0

	sc.Given().Command(newTaskCmd("task.create", ref.ID)).
		WhenQuery(EchoQuery{Value: "probe"}).
		ThenSuccess().
		ThenQueryEventuallyFails(func() (any, error) {
			calls++
			if calls < 3 {
				return "still there", nil
			}

			return nil, errGone
		}, errGone)
}

// TestThenQueryEventuallyFails_TimeoutReportsLastOutcome.
func TestThenQueryEventuallyFails_TimeoutReportsLastOutcome(t *testing.T) {
	t.Parallel()

	messages := runExpectingFailure(t, func(tb testing.TB) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ref := id.NewStreamRef("Task", id.NewStreamID())
		sc := systemscenario.System(tb, ctx, taskDomain(), memoryDeployment(),
			systemscenario.WithAwaitTimeout(60*time.Millisecond))

		sc.Given().Command(newTaskCmd("task.create", ref.ID)).
			WhenQuery(EchoQuery{Value: "probe"}).
			ThenQueryEventuallyFails(func() (any, error) { return "still there", nil },
				errors.New("task.gone"))
	})

	if len(messages) == 0 {
		t.Fatal("expected a timeout failure, got none")
	}

	last := messages[len(messages)-1]
	if !contains(last, "query still succeeds") || !contains(last, "still there") {
		t.Fatalf("timeout message must report the last non-matching outcome, got: %s", last)
	}
}

// ── WithQuietWindow (F17.1/F17.2) ──

// TestThenNoEvents_QuietWindowPassesAfterWindow: under await mode the
// negative assertion actually WATCHES — it passes only after the window
// elapses, not the instant it is called.
func TestThenNoEvents_QuietWindowPassesAfterWindow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sc := systemscenario.System(t, ctx, taskDomain(), memoryDeployment(),
		systemscenario.WithQuietWindow(120*time.Millisecond))

	start := time.Now()
	sc.TimeAdvances(time.Minute).ThenNoEvents()
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("ThenNoEvents passed after %s — the quiet window never ran", elapsed)
	}
}

// TestThenNoEvents_QuietWindowFailsOnTimerEvent: an event arriving inside
// the window fails the assertion — the previous instant-pass behavior could
// never catch a late timer.
func TestThenNoEvents_QuietWindowFailsOnTimerEvent(t *testing.T) {
	t.Parallel()

	messages := runExpectingFailure(t, func(tb testing.TB) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ref := id.NewStreamRef("Task", id.NewStreamID())
		sc := systemscenario.System(tb, ctx, timerDomain(ref), timerDeployment(),
			systemscenario.WithQuietWindow(2*time.Second))

		sc.Given(
			sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
		).TimeAdvances(2 * time.Hour).
			ThenNoEvents()
	})

	if len(messages) == 0 {
		t.Fatal("expected ThenNoEvents to fail on the fired timer, got none")
	}

	last := messages[len(messages)-1]
	if !contains(last, "task.updated") {
		t.Fatalf("failure must name the event that broke the quiet window, got: %s", last)
	}
}

// ── WithCommandCaptureFilter (F17.3) ──

// TestWithCommandCaptureFilter_ExcludesNoise: a filtered command is never
// captured — ThenCommands sees an empty act log while the journal still
// records the event.
func TestWithCommandCaptureFilter_ExcludesNoise(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(t, ctx, taskDomain(), memoryDeployment(),
		systemscenario.WithCommandCaptureFilter(func(cmd command.Command) bool {
			return cmd.Type() != "task.create"
		}))

	sc.When(newTaskCmd("task.create", ref.ID)).
		Then("task.created").
		ThenCommands().
		ThenCommandsSatisfy(func(cmds []command.Command) {
			if len(cmds) != 0 {
				t.Fatalf("filtered task.create must not be captured, got %d command(s)", len(cmds))
			}
		})
}

// contains keeps the diagnostic assertions terse.
func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
