package signing_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestExtractMultiSignature(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	t.Run("extract from unsigned event", func(t *testing.T) {
		t.Parallel()
		if _, err := signing.ExtractMultiSignature(evt); err == nil {
			t.Fatal("expected error for unsigned event")
		}
	})

	t.Run("extract from nil event", func(t *testing.T) {
		t.Parallel()
		if _, err := signing.ExtractMultiSignature(nil); err == nil {
			t.Fatal("expected error for nil event")
		}
	})

	t.Run("has multi-sig", func(t *testing.T) {
		t.Parallel()
		if signing.HasMultiSignature(evt) {
			t.Fatal("original event should not have multi-sig")
		}

		clone, _ := deviceMulti.Sign(evt)
		if !signing.HasMultiSignature(clone) {
			t.Fatal("signed event should have multi-sig")
		}
	})
}

func TestVerifyAll_MissingVerifier(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)

	verifiers := map[signing.Actor]signing.Verifier{}
	err := signing.VerifyAll(clone, verifiers)
	if err == nil {
		t.Fatal("expected error for missing verifier")
	}
}

func TestVerifyAll_FailingVerifier(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, _ := deviceMulti.Sign(evt)
	tampered, tamperErr := event.NewEvent(
		clone.Type(),
		clone.AggregateID(),
		clone.AggregateType(),
		clone.Version(),
		[]byte(`{"tampered":true}`),
		event.WithEventID(clone.ID()),
		event.WithOccurredAt(clone.OccurredAt()),
		event.WithSchemaVersion(clone.SchemaVersion()),
		event.WithMetadata(clone.Metadata()),
	)
	if tamperErr != nil {
		t.Fatalf("tamper: %v", tamperErr)
	}

	pubKey, _, _ := ed25519.GenerateKey(nil)
	verifier, _ := signing.NewEd25519Verifier(pubKey)

	verifiers := map[signing.Actor]signing.Verifier{signing.Actor("device"): verifier}
	err := signing.VerifyAll(tampered, verifiers)
	if err == nil {
		t.Fatal("expected error for tampered event with wrong verifier")
	}
}

func TestExtractMultiSignature_InvalidJSON(t *testing.T) {
	t.Parallel()

	// Create an event with malformed multi-sig JSON in metadata
	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evt, err := event.NewEvent(
		"test.invalid", aggID, "Test", 1, []byte(`{}`),
		event.WithMetadata(&event.Metadata{
			Custom: map[event.MetadataKey]string{
				signing.MultiSigMetadataKey: `{invalid json`,
			},
		}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	_, extractErr := signing.ExtractMultiSignature(evt)
	if extractErr == nil {
		t.Fatal("expected error for invalid JSON in multi-sig metadata")
	}
}

func TestVerifyAll_NilEvent(t *testing.T) {
	t.Parallel()

	verifiers := map[signing.Actor]signing.Verifier{}
	if err := signing.VerifyAll(nil, verifiers); err == nil {
		t.Fatal("expected error for nil event")
	}
}
