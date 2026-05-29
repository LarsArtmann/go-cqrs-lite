package signing_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/signing"
	"github.com/larsartmann/go-cqrs-lite/signing/multisig"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func subscribeTo(t *testing.T, bus *memory.MemoryBus, topic string, received *[]event.Event) {
	t.Helper()
	if err := bus.Subscribe(event.Type(topic), func(_ context.Context, evt event.Event) error {
		*received = append(*received, evt)

		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
}

// TestSigningFullFlow tests the complete signing pipeline across modules:
//
//	event.NewEvent -> MultiSigner.Sign -> Bus.Publish (with MultiSignMiddleware)
//	-> Bus.Subscribe (with RequireMultiSigMiddleware) -> verified handler
func TestSigningFullFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bus := memory.NewMemoryBus()
	defer bus.Close() //nolint:errcheck // test helper

	deviceKey := []byte("device-secret-key-thirty-two-by!")
	deviceHMAC, _ := signing.NewHMAC(deviceKey)
	serverKey := []byte("server-secret-key-thirty-two-by!")
	serverHMAC, _ := signing.NewHMAC(serverKey)

	deviceMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmHMACSHA256,
		deviceHMAC,
	)
	serverMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("server"),
		multisig.AlgorithmHMACSHA256,
		serverHMAC,
	)

	bus.UsePublish(multisig.MultiSignMiddleware(deviceMulti))
	bus.UsePublish(multisig.MultiSignMiddleware(serverMulti))

	verifiers := multisig.VerifierMap(deviceMulti, serverMulti)
	bus.Use(multisig.RequireMultiSigMiddleware(verifiers))

	var received []event.Event

	subscribeTo(t, bus, "user.created", &received)

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(
		"user.created",
		aggID,
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(ctx, evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if len(received) != 1 {
		t.Fatalf("expected 1 received event, got %d", len(received))
	}

	receivedEvt := received[0]

	if !multisig.HasMultiSignature(receivedEvt) {
		t.Fatal("received event should have multi-signature")
	}

	multiSig, err := multisig.ExtractMultiSignature(receivedEvt)
	if err != nil {
		t.Fatalf("extract multi-sig: %v", err)
	}

	if multiSig.Count() != 2 {
		t.Fatalf("expected 2 signatures, got %d", multiSig.Count())
	}

	if !multiSig.HasActor(multisig.Actor("device")) {
		t.Fatal("expected device signature")
	}

	if !multiSig.HasActor(multisig.Actor("server")) {
		t.Fatal("expected server signature")
	}

	if err := multisig.VerifyAll(receivedEvt, verifiers); err != nil {
		t.Fatalf("verify all: %v", err)
	}
}

// TestSigningTamperDetection tests that tampered events are rejected
// by the RequireMultiSigMiddleware.
func TestSigningTamperDetection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bus := memory.NewMemoryBus()
	defer bus.Close() //nolint:errcheck // test helper

	deviceKey := []byte("tamper-device-key-thirty-two-by!")
	deviceHMAC, _ := signing.NewHMAC(deviceKey)
	serverKey := []byte("tamper-server-key-thirty-two-by!")
	serverHMAC, _ := signing.NewHMAC(serverKey)

	deviceMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("device"),
		multisig.AlgorithmHMACSHA256,
		deviceHMAC,
	)
	serverMulti, _ := multisig.NewMultiSigner(
		multisig.Actor("server"),
		multisig.AlgorithmHMACSHA256,
		serverHMAC,
	)

	verifiers := multisig.VerifierMap(deviceMulti, serverMulti)
	bus.Use(multisig.RequireMultiSigMiddleware(verifiers))

	var received []event.Event

	subscribeTo(t, bus, "user.created", &received)

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(
		"user.created", aggID, "User", 1,
		[]byte(`{"name":"Alice"}`),
	)

	deviceSigned, _ := deviceMulti.Sign(evt)
	serverSigned, _ := serverMulti.Sign(deviceSigned)

	tampered := testhelpers.TamperEvent(serverSigned, []byte(`{"name":"Bob"}`))

	if err := bus.Publish(ctx, tampered); err == nil {
		t.Fatal("expected error for tampered event, got nil")
	}

	if len(received) != 0 {
		t.Fatalf("expected 0 received events (tampered), got %d", len(received))
	}
}
