package systemscenario_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

func TestSystem_ThenGoldenTrail(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event(
			"task.created",
			ref,
			TaskCreated{ID: ref.ID.String(), Title: "golden", Status: "pending"},
		),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated").
		ThenGolden(t, "golden_task_lifecycle")
}

func TestTrailIsDeterministicShape(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		ThenEvents(func(events []event.Event) {
			want := "- task.updated v2 actor=harness-fixture\n"
			if got := systemscenario.Trail(events); got != want {
				t.Fatalf("Trail:\nwant %q\ngot  %q", want, got)
			}
		})
}

// TestTrailWithPayloads_IsRunStable pins the F73 contract: payloads embed
// the minted stream ID, yet the masked hash is byte-identical across runs
// and the masked token actually replaced the ID.
func TestTrailWithPayloads_IsRunStable(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) string {
		t.Helper()

		sc, ref, _ := newTaskScenario(t)
		var trail string

		sc.Given(
			sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
		).When(newTaskCmd("task.complete", ref.ID)).
			ThenEvents(func(events []event.Event) {
				trail = systemscenario.TrailWithPayloads(events, []string{ref.ID.String()})
			})

		return trail
	}

	first, second := build(t), build(t)
	if first != second {
		t.Fatalf("TrailWithPayloads not run-stable:\nfirst:\n%ssecond:\n%s", first, second)
	}

	if !strings.Contains(first, "<<stream#0>>") {
		t.Fatalf("expected masked stream token in trail:\n%s", first)
	}

	if strings.Contains(first, "payload=sha256") || !strings.Contains(first, "payload=") {
		t.Fatalf("expected payload= suffix per line:\n%s", first)
	}
}

// TestTrailWithPayloads_MasksAppliedBeforeHash proves the hash covers MASKED
// bytes: two runs whose raw payloads differ only in the minted ID produce
// identical hashes, and an unmasked run differs from the masked one.
func TestTrailWithPayloads_MasksAppliedBeforeHash(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)
	var masked string

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		ThenEvents(func(events []event.Event) {
			masked = systemscenario.TrailWithPayloads(events, []string{ref.ID.String()})
			raw := systemscenario.TrailWithPayloads(events, nil)

			sameID := strings.Split(strings.Split(raw, "payload=")[1], "\n")[0]
			maskedID := strings.Split(strings.Split(masked, "payload=")[1], "\n")[0]
			if sameID != maskedID {
				t.Fatalf("masking with the true ID must not change the hash:\nraw %s\nmasked %s", sameID, maskedID)
			}
		})
}

func TestSystem_ThenGoldenWithPayloads(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Title: "golden", Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		Then("task.updated").
		ThenGoldenWithPayloads(t, "golden_task_payloads")
}
