package schema

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func driftTestEvents(
	t *testing.T,
	declared EventSchema,
	stampedFingerprint string,
) (matching, mismatched, unstamped event.Event) {
	t.Helper()

	matching = newPayloadEvent(t, declared.eventType, 2, []byte(`{}`))
	if stampedFingerprint != "" {
		var err error
		matching, err = rebuildWithMetadataStamp(
			matching,
			FingerprintMetadataKey,
			declared.Fingerprint(),
		)
		if err != nil {
			t.Fatalf("stamp matching: %v", err)
		}
	}

	mismatched = newPayloadEvent(t, declared.eventType, 2, []byte(`{}`))
	var err error
	mismatched, err = rebuildWithMetadataStamp(
		mismatched,
		FingerprintMetadataKey,
		"v1:stalefingerprint",
	)
	if err != nil {
		t.Fatalf("stamp mismatched: %v", err)
	}

	unstamped = newPayloadEvent(t, declared.eventType, 2, []byte(`{}`))

	return matching, mismatched, unstamped
}

func TestFingerprintDrift_AdvisoryHookFiresOnlyOnMismatch(t *testing.T) {
	t.Parallel()

	declared := Event("user.created", 2,
		RenameField("user.created", 1, "name", "displayName"),
	)

	matching, mismatched, unstamped := driftTestEvents(t, declared, "stamp")

	var findings []string

	transform := FingerprintDrift(
		[]EventSchema{declared},
		func(evt event.Event, stamped, declaredFp string) {
			findings = append(findings, evt.Type().String()+"="+stamped+"!="+declaredFp)
		},
	)

	got, err := transform([]event.Event{matching, mismatched, unstamped})
	if err != nil {
		t.Fatalf("drift transform must stay advisory: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("advisory transform must pass events through, got %d", len(got))
	}

	if len(findings) != 1 {
		t.Fatalf("expected exactly one drift finding, got %v", findings)
	}

	if findings[0] != "user.created=v1:stalefingerprint!="+declared.Fingerprint() {
		t.Fatalf("finding carried wrong details: %s", findings[0])
	}
}

func TestFingerprintDriftHard_FailsOnlyOnMismatch(t *testing.T) {
	t.Parallel()

	declared := Event("user.created", 2,
		RenameField("user.created", 1, "name", "displayName"),
	)

	matching, mismatched, unstamped := driftTestEvents(t, declared, "stamp")

	transform := FingerprintDriftHard([]EventSchema{declared})

	if _, err := transform([]event.Event{matching, unstamped}); err != nil {
		t.Fatalf("matching and unstamped events must pass hard mode: %v", err)
	}

	_, err := transform([]event.Event{mismatched})
	if err == nil {
		t.Fatal("mismatched event must fail hard mode")
	}

	if errorfamily.Classify(err) != errorfamily.Corruption {
		t.Fatalf("hard-mode drift is Corruption, got %v", errorfamily.Classify(err))
	}
}

func TestFingerprintDrift_UndeclaredTypeSilent(t *testing.T) {
	t.Parallel()

	declared := Event("user.created", 2)

	_, _, unstamped := driftTestEvents(t, declared, "")

	// An event of a type the declaration set does not mention carries a
	// foreign stamp — still silent: nothing is declared to compare against.
	foreign, err := rebuildWithMetadataStamp(
		newPayloadEvent(t, "user.unrelated", 1, []byte(`{}`)),
		FingerprintMetadataKey, "v1:someotherdecl",
	)
	if err != nil {
		t.Fatalf("stamp foreign: %v", err)
	}

	hookFired := false

	transform := FingerprintDrift([]EventSchema{declared}, func(event.Event, string, string) {
		hookFired = true
	})

	if _, err := transform([]event.Event{unstamped, foreign}); err != nil {
		t.Fatalf("transform: %v", err)
	}

	if hookFired {
		t.Fatal("undeclared type must not fire the drift hook")
	}
}
