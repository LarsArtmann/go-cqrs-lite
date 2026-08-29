package listing_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestStatus_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status listing.Status
		want   string
	}{
		{listing.StatusActive, "active"},
		{listing.StatusTombstoned, "tombstoned"},
		{listing.StatusUndetermined, "undetermined"},
		{listing.Status(99), "Status(99)"},
	}

	for _, tc := range cases {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("Status(%d).String() = %q, want %q", int(tc.status), got, tc.want)
		}
	}
}

func TestStatus_Predicates(t *testing.T) {
	t.Parallel()

	active := listing.StatusActive
	if !active.IsActive() || active.IsTombstoned() || !active.IsKnown() {
		t.Error("StatusActive predicates")
	}

	if listing.StatusTombstoned.IsActive() || !listing.StatusTombstoned.IsTombstoned() {
		t.Error("StatusTombstoned predicates")
	}

	if listing.StatusUndetermined.IsKnown() {
		t.Error("StatusUndetermined must not be known")
	}
}

func TestStatusClassifier_ClassifyLast(t *testing.T) {
	t.Parallel()

	classifier := listing.NewStatusClassifier(
		[]event.Type{"user.deleted", "order.cancelled"},
		[]event.Type{"user.reactivated"},
	)

	cases := []struct {
		name      string
		eventType event.Type
		want      listing.Status
	}{
		{"delete type", "user.deleted", listing.StatusTombstoned},
		{"second delete type", "order.cancelled", listing.StatusTombstoned},
		{"rebirth type", "user.reactivated", listing.StatusActive},
		{"ordinary type", "user.renamed", listing.StatusActive},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			evt, err := event.NewEvent(
				tc.eventType, id.NewStreamID(), "User",
				event.Version(1), []byte(`{}`),
			)
			if err != nil {
				t.Fatal(err)
			}

			if got := classifier.ClassifyLast(evt); got != tc.want {
				t.Errorf("ClassifyLast(%s) = %v, want %v", tc.eventType, got, tc.want)
			}
		})
	}

	t.Run("zero-value classifier is undetermined", func(t *testing.T) {
		t.Parallel()

		evt, err := event.NewEvent(
			"user.deleted", id.NewStreamID(), "User",
			event.Version(1), []byte(`{}`),
		)
		if err != nil {
			t.Fatal(err)
		}

		if got := (listing.StatusClassifier{}).ClassifyLast(
			evt,
		); got != listing.StatusUndetermined {
			t.Errorf("zero classifier = %v, want StatusUndetermined", got)
		}
	})
}

// TestStatusClassifier_ParityWithMetadataTombstones verifies the type-driven
// classifier produces the same Status the legacy metadata bridge
// (event.MarkTombstone + event.DetectTombstone) produced for the delete and
// rebirth scenarios (ADR-0114 migration parity).
func TestStatusClassifier_ParityWithMetadataTombstones(t *testing.T) {
	t.Parallel()

	classifier := listing.NewStatusClassifier(
		[]event.Type{"user.deleted"},
		[]event.Type{"user.reactivated"},
	)

	newEvent := func(eventType event.Type) event.Event {
		t.Helper()

		evt, err := event.NewEvent(
			eventType, id.NewStreamID(), "User",
			event.Version(1), []byte(`{}`),
		)
		if err != nil {
			t.Fatal(err)
		}

		return evt
	}

	scenarios := []struct {
		name string
		last event.Event
		want listing.Status
	}{
		{"deleted stream", newEvent("user.deleted"), listing.StatusTombstoned},
		{"reborn stream", newEvent("user.reactivated"), listing.StatusActive},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := classifier.ClassifyLast(tc.last); got != tc.want {
				t.Fatalf("classifier = %v, want %v", got, tc.want)
			}

			var legacy event.TombstoneStatus

			var err error

			//nolint:exhaustive // undetermined is the empty-stream value
			switch tc.want {
			case listing.StatusTombstoned:
				var marked *event.ImmutableEvent
				marked, err = event.MarkTombstone(tc.last)
				if err != nil {
					t.Fatal(err)
				}

				legacy = event.DetectTombstone([]event.Event{marked})
			case listing.StatusActive:
				var marked *event.ImmutableEvent
				marked, err = event.MarkRebirth(tc.last)
				if err != nil {
					t.Fatal(err)
				}

				legacy = event.DetectTombstone([]event.Event{marked})
			}

			if int(legacy) != int(tc.want) {
				t.Errorf(
					"legacy metadata status = %v (%d), type-driven = %v (%d): wire values must match",
					legacy,
					int(legacy),
					tc.want,
					int(tc.want),
				)
			}
		})
	}
}

// TestInMemoryStreamReader_NoClassifierMatchesLegacyUnmarked verifies that a
// reader without a classifier reports StatusUndetermined — the same value
// the metadata bridge returned for unmarked streams.
func TestInMemoryStreamReader_NoClassifierMatchesLegacyUnmarked(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	reader := listing.NewInMemoryStreamReader(store)

	evt, err := event.NewEvent(
		"user.created", id.NewStreamID(), "User",
		event.Version(1), []byte(`{"name":"Alice"}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	streamID := evt.StreamID()

	if err = store.Save(
		t.Context(), id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt}, event.Version(0),
	); err != nil {
		t.Fatal(err)
	}

	page, err := reader.ListWithStatus(t.Context(), listing.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}

	if got := page.Items[0].Status; got != listing.StatusUndetermined {
		t.Errorf("no-classifier status = %v, want StatusUndetermined", got)
	}
}
