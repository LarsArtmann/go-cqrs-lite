package schema

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func TestFingerprintStableAcrossRuns(t *testing.T) {
	t.Parallel()

	// The fingerprint is a pure function of the declaration: two identically
	// declared schemas (constructed separately, order of op list equal after
	// sorting) must agree — across processes too, since nothing seeded or
	// random rides along.
	first := Event("user.created", 2,
		RenameField("user.created", 1, "name", "displayName"),
		AddField("user.created", 1, "rank", 0),
	)
	second := Event("user.created", 2,
		RenameField("user.created", 1, "name", "displayName"),
		AddField("user.created", 1, "rank", 0),
	)

	if first.Fingerprint() != second.Fingerprint() {
		t.Fatalf("identical declarations fingerprinted differently: %s vs %s",
			first.Fingerprint(), second.Fingerprint())
	}

	if len(first.Fingerprint()) == 0 || first.Fingerprint()[:3] != "v1:" {
		t.Fatalf("fingerprint not versioned: %s", first.Fingerprint())
	}
}

func TestFingerprintDiffersOnShapeChange(t *testing.T) {
	t.Parallel()

	base := Event("user.created", 2,
		RenameField("user.created", 1, "name", "displayName"),
	)
	baseFingerprint := base.Fingerprint()

	changed := []EventSchema{
		// Current version bump.
		Event("user.created", 3,
			RenameField("user.created", 1, "name", "displayName"),
		),
		// Op added.
		Event("user.created", 2,
			RenameField("user.created", 1, "name", "displayName"),
			AddField("user.created", 1, "rank", 0),
		),
		// Op removed.
		Event("user.created", 2),
		// Op re-versioned.
		Event("user.created", 2,
			RenameField("user.created", 2, "name", "displayName"),
		),
		// Different event type entirely.
		Event("user.deleted", 2,
			RenameField("user.deleted", 1, "name", "displayName"),
		),
	}

	for i, drifted := range changed {
		if drifted.Fingerprint() == baseFingerprint {
			t.Errorf("case %d: shape change did not change the fingerprint", i)
		}
	}
}

func TestFingerprintStampWritePath(t *testing.T) {
	t.Parallel()

	declared := []EventSchema{
		Event("user.created", 2,
			RenameField("user.created", 1, "name", "displayName"),
		),
	}

	stamp := FingerprintStamp(declared, FingerprintMetadataKey)

	declaredEvent := newPayloadEvent(t, "user.created", 2, []byte(`{"displayName":"Lars"}`))
	undeclaredEvent := newPayloadEvent(t, "user.unrelated", 1, []byte(`{}`))

	got, err := stamp([]event.Event{declaredEvent, undeclaredEvent})
	if err != nil {
		t.Fatalf("stamp: %v", err)
	}

	stampedFingerprint := got[0].Metadata().Custom[event.MetadataKey(FingerprintMetadataKey)]
	if stampedFingerprint == "" {
		t.Fatal("declared event was not stamped")
	}

	if stampedFingerprint != declared[0].Fingerprint() {
		t.Fatalf("stamped %s, want declaration fingerprint %s",
			stampedFingerprint, declared[0].Fingerprint())
	}

	if got[0].ID() != declaredEvent.ID() {
		t.Fatal("stamping must preserve the event ID")
	}

	if !reflect.DeepEqual(got[0].Payload(), declaredEvent.Payload()) {
		t.Fatal("stamping must not touch the payload bytes")
	}

	if got[1].Metadata().Custom[event.MetadataKey(FingerprintMetadataKey)] != "" {
		t.Fatal("undeclared event must pass through unstamped")
	}

	// Idempotence: re-stamping stamped events is a no-op (same instance back).
	again, err := stamp(got)
	if err != nil {
		t.Fatalf("re-stamp: %v", err)
	}

	if again[0] != got[0] {
		t.Fatal("already-stamped event must be returned unchanged")
	}
}
