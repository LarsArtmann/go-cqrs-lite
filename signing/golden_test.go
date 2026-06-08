package signing_test

import (
	"encoding/json"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestGolden_HMACSignedEvent(t *testing.T) {
	key := []byte("golden-test-key-exactly-32-bytes!")
	signer, err := signing.NewHMAC(key)
	if err != nil {
		t.Fatalf("create HMAC signer: %v", err)
	}

	evt := fixedSignEvent(t)
	sig, err := signer.Sign(evt)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	signed, err := signing.AttachSignature(evt, sig)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	got, err := json.MarshalIndent(signed.Metadata(), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "hmac-signed-metadata.json"),
		got,
		*updateGolden,
	)
}

func TestGolden_SignatureJSONEncoding(t *testing.T) {
	sig := signing.Signature([]byte("deterministic-signature-for-golden-test"))

	got, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "signature-json.json"),
		got,
		*updateGolden,
	)
}

func fixedSignEvent(t *testing.T) event.Event {
	t.Helper()

	aggID := id.MustParseAggregateID("01HK1540X0841Y0A6BSX1VKR95")
	evtID := id.MustParseEventID("01HK1540X0841Y0A6BSX1VKR96")

	evt, err := event.NewEvent(
		"order.created", aggID, "Order", 1,
		[]byte(`{"item":"widget","quantity":3}`),
		event.WithEventID(evtID),
		event.WithOccurredAt(time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)),
		event.WithSchemaVersion(2),
		event.WithCorrelationID(id.MustParseCorrelationID("01HK1540X0841Y0A6BSX1VKR97")),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	return evt
}
