package signing_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3/multisig"
)

func subscribeTo(t *testing.T, bus *eventtest.FakeBus, topic string, received *[]event.Event) {
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
	bus := eventtest.NewFakeBus()
	defer bus.Close()

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

	_ = bus.UsePublish(multisig.MultiSignMiddleware(deviceMulti))
	_ = bus.UsePublish(multisig.MultiSignMiddleware(serverMulti))

	verifiers, err := multisig.VerifierMap(deviceMulti, serverMulti)
	if err != nil {
		t.Fatal(err)
	}

	_ = bus.Use(multisig.RequireMultiSigMiddleware(verifiers))

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
	bus := eventtest.NewFakeBus()
	defer bus.Close()

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

	verifiers, err := multisig.VerifierMap(deviceMulti, serverMulti)
	if err != nil {
		t.Fatal(err)
	}

	_ = bus.Use(multisig.RequireMultiSigMiddleware(verifiers))

	var received []event.Event

	subscribeTo(t, bus, "user.created", &received)

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent(
		"user.created", aggID, "User", 1,
		[]byte(`{"name":"Alice"}`),
	)

	deviceSigned, _ := deviceMulti.Sign(evt)
	serverSigned, _ := serverMulti.Sign(deviceSigned)

	tampered := eventtest.TamperEvent(serverSigned, []byte(`{"name":"Bob"}`))

	if err := bus.Publish(ctx, tampered); err == nil {
		t.Fatal("expected error for tampered event, got nil")
	}

	if len(received) != 0 {
		t.Fatalf("expected 0 received events (tampered), got %d", len(received))
	}
}
