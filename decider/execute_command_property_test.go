package decider_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestExecuteCommandRefCausationProperty verifies across arbitrary command
// identities, event counts, and decide-side causation collisions that the
// stamp lands exactly on the events that did not already carry one.
func TestExecuteCommandRefCausationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		store := memory.NewMemoryStore()

		repo, err := decider.NewRepository(store, nil, auditedCounterDecider())
		if err != nil {
			t.Fatalf("repository: %v", err)
		}

		ref := id.NewStreamRef("Counter", id.NewStreamID())

		eventCount := rapid.IntRange(0, 4).Draw(t, "eventCount")
		cmdID := id.NewCommandID()
		zeroID := rapid.Bool().Draw(t, "zeroID")
		preStampedIdx := rapid.IntRange(-1, 4).Draw(t, "preStampedIdx")

		if zeroID {
			cmdID = id.CommandID{}
		}

		cmd := &auditedCommand{id: cmdID, cmdType: "PropCommand"}

		err = decider.ExecuteCommandRef(ctx, repo, ref, cmd,
			func(_ int, v event.Version, _ *auditedCommand) ([]event.Event, error) {
				evts := make([]event.Event, 0, eventCount)
				for i := range eventCount {
					if i == preStampedIdx {
						kept, keptErr := event.NewEvent("CounterCreated", ref.ID, "Counter",
							v+event.Version(i)+1, []byte(`{}`),
							event.WithCausation("DecideSet", id.NewCommandID()))
						if keptErr != nil {
							return nil, keptErr
						}

						evts = append(evts, kept)

						continue
					}

					evts = append(
						evts,
						makeCounterEvent("CounterIncremented", ref.ID, v+event.Version(i)+1),
					)
				}

				return evts, nil
			})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}

		evts, readErr := store.ReadAll(ctx)
		if readErr != nil {
			t.Fatalf("read: %v", readErr)
		}

		if len(evts) != eventCount {
			t.Fatalf("persisted %d events, want %d", len(evts), eventCount)
		}

		verifyCausationStamps(t, evts, cmdID, preStampedIdx)
	})
}

// verifyCausationStamps asserts the stamp outcome for every persisted event:
// decide-set causation preserved, zero-ID commands unstamped, everything
// else carrying the command's causation (typed form + record Cause).
func verifyCausationStamps(
	t *testing.T,
	evts []event.Event,
	cmdID id.CommandID,
	preStampedIdx int,
) {
	t.Helper()

	for i, evt := range evts {
		md := evt.Metadata()

		if i == preStampedIdx {
			if md.Causation == nil || md.Causation.CommandType != "DecideSet" {
				t.Fatalf("event %d: decide-set causation clobbered", i)
			}

			continue
		}

		if cmdID.IsZero() {
			if md.Causation != nil {
				t.Fatalf("event %d: stamped despite zero command ID", i)
			}

			continue
		}

		rec := event.AsRecord(evt)
		if md.Causation == nil ||
			md.Causation.CommandID != cmdID ||
			rec.MetaData.Cause.Kind != record.CauseCommand ||
			rec.MetaData.Cause.ID != cmdID.String() {
			t.Fatalf("event %d: causation stamp missing or wrong", i)
		}
	}
}
