package systemscenario_test

// Read-side depth specs: poison events dead-lettering through
// projectionhost.WithDeadLetterStore, and given-phase upcasting through
// DomainConfig.Schema (TODO_LIST "BDD harness adoption wave" residue —
// 02-46 §f22/§f23).

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// ── Dead-letter queue (02-46 §f22) ──

// TestDepth_PoisonEventDeadLetters proves the poison-message story end to
// end: an event whose payload cannot decode into the projection's declared
// struct exhausts its attempts and lands in the configured DeadLetterStore
// with diagnostics, while the healthy events around it still fold.
func TestDepth_PoisonEventDeadLetters(t *testing.T) {
	t.Parallel()

	dlq := projectionhost.NewMemoryDeadLetterStore()

	domain := taskDomain()
	domain.ProjectionHostOptions = append(
		domain.ProjectionHostOptions,
		projectionhost.WithDeadLetterStore(dlq, 1),
	)

	sc, ref, ctx := newDepthScenario(t, domain)

	poisonRef := id.NewStreamRef("Task", id.NewStreamID())

	sc.Given(
		// A second, poisoned stream: created folds, then a task.updated whose
		// Title is a number cannot decode into TaskUpdated.
		sc.Event("task.created", poisonRef, TaskCreated{
			ID: poisonRef.ID.String(), Title: "poisoned stream", Status: "pending",
		}),
		sc.Event("task.updated", poisonRef, map[string]any{
			"ID":     poisonRef.ID.String(),
			"Title":  42,
			"Status": "completed",
		}),
	).When(newTaskCmd("task.rename", ref.ID)).
		Await().
		ThenQuery(taskViewQuery(sc, ctx, ref.ID.String()),
			TaskView{ID: ref.ID.String(), Title: "renamed", Status: "pending"}).
		ThenQueryFunc(func() (any, error) { return dlq.List(ctx, "") }, func(got any) error {
			entries, ok := got.([]projectionhost.DeadLetterEntry)
			if !ok {
				return fmt.Errorf("want []projectionhost.DeadLetterEntry, got %T", got)
			}

			if len(entries) != 1 {
				return fmt.Errorf("want exactly 1 dead-letter entry, got %d", len(entries))
			}

			entry := entries[0]
			if entry.EventType != "task.updated" {
				return fmt.Errorf("dead-lettered event type: want task.updated, got %s", entry.EventType)
			}

			if entry.StreamID != poisonRef.ID.String() {
				return fmt.Errorf("dead-lettered stream: want %s, got %s",
					poisonRef.ID.String(), entry.StreamID)
			}

			if entry.ProjectionName == "" || entry.EventID == "" || entry.Error == "" {
				return fmt.Errorf("dead-letter entry missing diagnostics: %+v", entry)
			}

			return nil
		})
}

// ── Given-phase upcasting (02-46 §f23) ──

// legacyTaskCreatedV1 is the v1 wire shape of TaskCreated (the title field
// was called Name before the v2 rename).
type legacyTaskCreatedV1 struct {
	ID     string
	Name   string
	Status string
}

// TestDepth_GivenPhaseUpcastsLegacyPayloads proves the schema-evolution
// story through the WHOLE system: DomainConfig.Schema declares the v1→v2
// field rename, a v1-stamped given event upcasts on every read path — decider
// fold AND the projection host's journal — so the legacy title flows into
// both the emitted event and the read model.
func TestDepth_GivenPhaseUpcastsLegacyPayloads(t *testing.T) {
	t.Parallel()

	domain := taskDomain()
	domain.Schema = []schema.EventSchema{
		schema.Event("task.created", 2,
			schema.RenameField("task.created", 1, "Name", "Title"),
		),
	}

	sc, ref, ctx := newDepthScenario(t, domain)

	phase := sc.Given(
		sc.Event("task.created", ref, legacyTaskCreatedV1{
			ID:     ref.ID.String(),
			Name:   "legacy title",
			Status: "pending",
		}, event.WithSchemaVersion(1)),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated")

	systemscenario.ThenPayload(phase, 0, TaskUpdated{
		ID:     ref.ID.String(),
		Title:  "legacy title",
		Status: "completed",
	})

	phase.ThenQuery(taskViewQuery(sc, ctx, ref.ID.String()),
		TaskView{ID: ref.ID.String(), Title: "legacy title", Status: "completed"})
}
