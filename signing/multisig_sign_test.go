package signing_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/signing"
)

func TestMultiSigner_SignAddsActor(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	extracted, err := signing.ExtractMultiSignature(clone)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 1 {
		t.Fatalf("expected 1 entry, got %d", extracted.Count())
	}
	if !extracted.HasActor(signing.Actor("device")) {
		t.Fatal("expected device actor")
	}
}

func TestMultiSigner_MultipleActors(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	serverMulti := newServerMultiSigner(t)
	evt := makeTestEvent(t)

	clone1, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("device sign: %v", err)
	}

	clone2, err := serverMulti.Sign(clone1)
	if err != nil {
		t.Fatalf("server sign: %v", err)
	}

	extracted, err := signing.ExtractMultiSignature(clone2)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extracted.Count() != 2 {
		t.Fatalf("expected 2 entries, got %d", extracted.Count())
	}
	if !extracted.HasActor(signing.Actor("device")) ||
		!extracted.HasActor(signing.Actor("server")) {
		t.Fatal("expected both device and server actors")
	}
}

func TestMultiSigner_ReSignReplaces(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)
	evt := makeTestEvent(t)

	clone1, err := deviceMulti.Sign(evt)
	if err != nil {
		t.Fatalf("first sign: %v", err)
	}

	extracted1, _ := signing.ExtractMultiSignature(clone1)
	entry1 := extracted1.Get(signing.Actor("device"))
	if entry1 == nil {
		t.Fatal("expected device entry after first sign")
	}

	clone2, err := deviceMulti.Sign(clone1)
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}

	extracted2, _ := signing.ExtractMultiSignature(clone2)
	if extracted2.Count() != 1 {
		t.Fatalf("expected 1 entry after re-sign, got %d", extracted2.Count())
	}

	entry2 := extracted2.Get(signing.Actor("device"))
	if entry2.SignedAt.Equal(entry1.SignedAt) {
		t.Fatal("re-signed entry should have different timestamp")
	}
}

func TestMultiSigner_NilEvent(t *testing.T) {
	t.Parallel()

	deviceMulti, _ := newDeviceMultiSigner(t)

	if _, err := deviceMulti.Sign(nil); err == nil {
		t.Fatal("expected error for nil event on sign")
	}

	if err := deviceMulti.Verify(nil); err == nil {
		t.Fatal("expected error for nil event on verify")
	}
}
