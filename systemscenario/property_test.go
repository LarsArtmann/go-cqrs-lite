package systemscenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"pgregory.net/rapid"
)

// TestProperty_RandomCommandSequencesKeepJournalOrdered drives a real
// system.New boot with rapid-generated command sequences: every sequence
// must leave each stream's journal strictly version-ordered, no matter how
// adversarial the order of create/rename/complete dispatches.
func TestProperty_RandomCommandSequencesKeepJournalOrdered(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sys, err := system.New(ctx, taskDomain(), memoryDeployment())
		if err != nil {
			rt.Fatalf("system.New: %v", err)
		}

		defer func() { _ = sys.Close() }()

		if err := sys.Start(ctx); err != nil {
			rt.Fatalf("Start: %v", err)
		}

		streamID := id.NewStreamID()
		_ = sys.CommandDispatcher().Dispatch(ctx, newTaskCmd("task.create", streamID))

		sequence := rapid.SliceOf(rapid.SampledFrom([]command.Type{
			"task.create", "task.rename", "task.complete",
		})).Draw(rt, "sequence")

		for _, cmdType := range sequence {
			_ = sys.CommandDispatcher().Dispatch(ctx, newTaskCmd(cmdType, streamID))
		}

		events, err := sys.EventStore().Load(ctx, id.NewStreamRef("Task", streamID))
		if err != nil {
			rt.Fatalf("load stream: %v", err)
		}

		last := event.Version(0)
		for _, evt := range events {
			if evt.Version() <= last {
				rt.Fatalf(
					"journal versions not strictly increasing: v%d after v%d",
					evt.Version(),
					last,
				)
			}

			last = evt.Version()
		}
	})
}

// TestProperty_ReadModelMatchesFold is the fold-equivalence invariant: for
// any random command sequence, the asynchronously projected read-model row
// must equal the pure fold over the stream's journal events (the decider's
// Apply semantics). A dropped, duplicated, or mis-projected event — anywhere
// between journal write and Lookup view — breaks the equality. Rejected
// commands (create-on-existing, complete-on-completed) are part of the
// generated adversarial surface; they simply leave no journal events, and
// the fold over the ACTUAL journal stays the oracle.
func TestProperty_ReadModelMatchesFold(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rapid.Check(t, func(rt *rapid.T) {
		// The harness allows exactly one Given phase per scenario, so each
		// generated sequence gets its own boot. Adopt keeps the lifecycle
		// caller-owned: defer Close tears the system down PER ITERATION
		// instead of piling 100 live systems until the rapid test ends.
		sys, err := system.New(ctx, taskDomain(), memoryDeployment())
		if err != nil {
			rt.Fatalf("system.New: %v", err)
		}

		defer func() { _ = sys.Close() }()

		if err := sys.Start(ctx); err != nil {
			rt.Fatalf("Start: %v", err)
		}

		scenario := systemscenario.Adopt(t, ctx, sys)

		streamID := id.NewStreamID()

		phase := scenario.Given().Command(newTaskCmd("task.create", streamID)).
			When(newTaskCmd("task.rename", streamID))

		sequence := rapid.SliceOf(rapid.SampledFrom([]command.Type{
			"task.create", "task.rename", "task.complete",
		})).Draw(rt, "sequence")

		for _, cmdType := range sequence {
			phase = phase.Command(newTaskCmd(cmdType, streamID))
		}

		events, err := scenario.System().EventStore().Load(ctx, id.NewStreamRef("Task", streamID))
		if err != nil {
			rt.Fatalf("load stream: %v", err)
		}

		state := TaskState{}
		for _, evt := range events {
			state, err = applyTask(state, evt)
			if err != nil {
				rt.Fatalf("fold event %s: %v", evt.Type(), err)
			}
		}

		phase.ThenQuery(
			taskViewQuery(scenario, ctx, streamID.String()),
			TaskView{ID: streamID.String(), Title: state.Title, Status: state.Status},
		)
	})
}

// TestSystem_JournalEquivalenceAcrossDeployments is the deployment-swap
// proof: the same command scenario on two independently booted deployments
// must produce observationally equal journals.
func TestSystem_JournalEquivalenceAcrossDeployments(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runOn := func() *system.System {
		sys, err := system.New(ctx, taskDomain(), memoryDeployment())
		if err != nil {
			t.Fatalf("system.New: %v", err)
		}

		t.Cleanup(func() { _ = sys.Close() })

		if err := sys.Start(ctx); err != nil {
			t.Fatalf("Start: %v", err)
		}

		streamID := id.NewStreamID()
		for _, cmdType := range []command.Type{"task.create", "task.rename", "task.complete"} {
			if err := sys.CommandDispatcher().
				Dispatch(ctx, newTaskCmd(cmdType, streamID)); err != nil {
				t.Fatalf("dispatch %s: %v", cmdType, err)
			}
		}

		return sys
	}

	systemscenario.AssertJournalEquivalence(t, ctx, runOn(), runOn())
}
