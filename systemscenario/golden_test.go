package systemscenario_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
// the minted stream ID (a fresh ULID every run), yet the masked hash is
// byte-identical across runs — and differs from the unmasked hash within a
// run, proving the substitution actually fired.
func TestTrailWithPayloads_IsRunStable(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) (string, string) {
		t.Helper()

		var masked, raw string

		sc, ref, _ := newTaskScenario(t)

		sc.Given(
			sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
		).When(newTaskCmd("task.complete", ref.ID)).
			ThenEvents(func(events []event.Event) {
				masked = systemscenario.TrailWithPayloads(events, []string{ref.ID.String()})
				raw = systemscenario.TrailWithPayloads(events, nil)
			})

		return masked, raw
	}

	maskedA, rawA := build(t)
	maskedB, rawB := build(t)

	if maskedA != maskedB {
		t.Fatalf("TrailWithPayloads not run-stable:\nfirst:\n%ssecond:\n%s", maskedA, maskedB)
	}

	if maskedA == rawA {
		t.Fatalf("masking with the true stream ID must change the hash:\n%s", maskedA)
	}

	_ = rawB
}

// TestTrailWithPayloads_MasksBeforeHash pins the mechanics: the rendered
// hash equals sha256 over the payload bytes with the ID manually replaced
// by its first-appearance token.
func TestTrailWithPayloads_MasksBeforeHash(t *testing.T) {
	t.Parallel()

	sc, ref, _ := newTaskScenario(t)
	var got, want string

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{ID: ref.ID.String(), Status: "pending"}),
	).When(newTaskCmd("task.complete", ref.ID)).
		ThenEvents(func(events []event.Event) {
			got = systemscenario.TrailWithPayloads(events, []string{ref.ID.String()})

			masked := bytes.ReplaceAll(
				events[0].Payload(),
				[]byte(ref.ID.String()),
				[]byte("<<stream#0>>"),
			)
			sum := sha256.Sum256(masked)
			want = hex.EncodeToString(sum[:])
		})

	rendered, _, _ := strings.Cut(strings.Split(got, "payload=")[1], "\n")
	if rendered != want {
		t.Fatalf("masked hash mismatch:\nwant %s\ngot  %s", want, rendered)
	}
}

func TestSystem_ThenGoldenWithPayloads(t *testing.T) {
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
		ThenGoldenWithPayloads(t, "golden_task_payloads")
}
