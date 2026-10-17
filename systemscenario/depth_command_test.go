package systemscenario_test

// Command-side and journal-side depth specs: idempotent redelivery,
// optimistic-concurrency conflicts, and Scenario.Event versioning under
// out-of-band appends (TODO_LIST "BDD harness adoption wave" residue —
// 14:40 §f13, 02-46 §f24/§f25).

import (
	"context"
	"fmt"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// newDepthScenario boots a task-domain variant on the memory deployment.
func newDepthScenario(
	t *testing.T,
	domain system.DomainConfig,
) (*systemscenario.Scenario, id.StreamRef, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ref := id.NewStreamRef("Task", id.NewStreamID())
	sc := systemscenario.System(t, ctx, domain, memoryDeployment())

	return sc, ref, ctx
}

// ── Idempotency (02-46 §f24) ──

// dedupOnCommandID makes command dispatch idempotent per command ID via the
// deployment's metaengine.DedupStore capability (the ADR-0158 migration
// target for idempotency/*): a replayed command is a success no-op, never a
// second handler run.
func dedupOnCommandID(sys *system.System) command.Middleware {
	dedup, err := sys.Dedup()
	if err != nil {
		panic(err)
	}

	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			seen, err := dedup.DedupCheckAndRecord(
				ctx, "depth.idempotency", cmd.ID().String(), time.Minute, time.Now())
			if err != nil {
				return err
			}

			if seen {
				return nil
			}

			return next(ctx, cmd)
		}
	}
}

// TestDepth_IdempotentCommandRedeliveryEmitsOneEvent proves the same-command-
// twice story: dispatching the SAME command (one ID) through a dedup-wired
// domain emits its event exactly once — the redelivery is a success no-op,
// so a retried transport cannot double-apply the decision.
func TestDepth_IdempotentCommandRedeliveryEmitsOneEvent(t *testing.T) {
	t.Parallel()

	domain := taskDomain()
	baseCommands := domain.Commands
	domain.Commands = func(sys *system.System) {
		baseCommands(sys)
		sys.UseCommandMiddleware(dedupOnCommandID(sys))
	}

	sc, ref, _ := newDepthScenario(t, domain)

	cmd := newTaskCmd("task.create", ref.ID)

	sc.Given().When(cmd).Then("task.created")

	// The redelivery: same command identity, dispatched again. It succeeds
	// as a no-op — and the act window (diffed against the FIRST act's
	// baseline, so both deliveries count) still holds exactly one event.
	sc.When(cmd).ThenSuccess().ThenEventsSatisfy(func(events []event.Event) error {
		if len(events) != 1 {
			return fmt.Errorf("want exactly the first delivery's event after redelivery, got %d", len(events))
		}

		return nil
	})
}

// ── Optimistic concurrency (02-46 §f25) ──

// registerRacedComplete registers task.complete_raced, whose decision
// appends a competing task.updated out-of-band between the decider's load
// and save — the deterministic replay of a racing writer winning the
// optimistic-concurrency window.
func registerRacedComplete(sys *system.System) {
	if err := system.RegisterCommand[*command.BasicCommand, TaskState](
		sys,
		"task.complete_raced",
		func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), "Task",
				func(state TaskState, version event.Version) ([]event.Event, error) {
					if !state.Exists {
						return nil, errTaskMissing
					}

					winner := taskEvent(
						"task.updated",
						cmd.StreamID(),
						version+1,
						TaskUpdated{
							ID:     cmd.StreamID().String(),
							Title:  "competing writer won",
							Status: state.Status,
						},
					)
					ref := id.NewStreamRef("Task", cmd.StreamID())
					if err := sys.EventStore().AppendBatch(ctx, ref, []event.Event{winner}); err != nil {
						return nil, err
					}

					// The stale decision: it still believes version+1 is
					// free, but the competitor just took it.
					return []event.Event{
						taskEvent(
							"task.updated",
							cmd.StreamID(),
							version+1,
							TaskUpdated{
								ID:     cmd.StreamID().String(),
								Title:  state.Title,
								Status: "completed",
							},
						),
					}, nil
				})
		},
	); err != nil {
		panic(err)
	}
}

// TestDepth_StaleExpectedVersionYieldsConflict proves the losing side of an
// optimistic-concurrency race: the stale save is rejected with a
// Conflict-family error, its event is NOT persisted, and only the competing
// writer's event lands in the journal.
func TestDepth_StaleExpectedVersionYieldsConflict(t *testing.T) {
	t.Parallel()

	domain := taskDomain()
	baseCommands := domain.Commands
	domain.Commands = func(sys *system.System) {
		baseCommands(sys)
		registerRacedComplete(sys)
	}

	sc, ref, _ := newDepthScenario(t, domain)

	phase := sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "race", Status: "pending"}),
	).When(newTaskCmd("task.complete_raced", ref.ID)).
		ThenErrorFamily(errorfamily.Conflict).
		Then("task.updated").
		ThenEventsSatisfy(func(events []event.Event) error {
			if len(events) != 1 {
				return fmt.Errorf("want exactly the competitor's event, got %d act events", len(events))
			}

			return nil
		})

	systemscenario.ThenPayload(phase, 0, TaskUpdated{
		ID:     ref.ID.String(),
		Title:  "competing writer won",
		Status: "pending",
	})
}

// ── Version hints under out-of-band appends (14:40 §f13) ──

// TestDepth_EventMintPicksUpOutOfBandAppends pins the Scenario.Event
// versioning contract under the System() escape hatch: an out-of-band
// AppendBatch between mints advances the stream, and the next mint continues
// from the store's latest version instead of a stale per-stream hint.
func TestDepth_EventMintPicksUpOutOfBandAppends(t *testing.T) {
	t.Parallel()

	sc, ref, ctx := newDepthScenario(t, taskDomain())

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "hinted", Status: "pending"}),
	)

	outOfBand := taskEvent("task.updated", ref.ID, 2, TaskUpdated{
		ID:     ref.ID.String(),
		Title:  "out-of-band",
		Status: "pending",
	})
	if err := sc.System().EventStore().AppendBatch(ctx, ref, []event.Event{outOfBand}); err != nil {
		t.Fatalf("out-of-band append: %v", err)
	}

	sc.WhenEvent(
		sc.Event("task.updated", ref, TaskUpdated{ID: ref.ID.String(), Title: "resumed", Status: "completed"}),
	).ThenEventsSatisfy(func(events []event.Event) error {
		if len(events) != 1 || events[0].Version() != 3 {
			return fmt.Errorf("want the act event at v3 (past the out-of-band v2), got %d event(s)", len(events))
		}

		return nil
	})
}
