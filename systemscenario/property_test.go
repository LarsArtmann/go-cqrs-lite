package systemscenario_test

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
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
				rt.Fatalf("journal versions not strictly increasing: v%d after v%d", evt.Version(), last)
			}

			last = evt.Version()
		}
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
			if err := sys.CommandDispatcher().Dispatch(ctx, newTaskCmd(cmdType, streamID)); err != nil {
				t.Fatalf("dispatch %s: %v", cmdType, err)
			}
		}

		return sys
	}

	systemscenario.AssertJournalEquivalence(t, ctx, runOn(), runOn())
}
