package watermill_test

import (
	"context"
	"encoding/json"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	wm "github.com/larsartmann/go-cqrs-lite/watermill/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func parseCausationID(s string) id.CausationID {
	v, err := id.ParseCausationID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func parseCorrID(s string) id.CorrelationID {
	v, err := id.ParseCorrelationID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func parseEventID(s string) id.EventID {
	v, err := id.ParseEventID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func parseUserID(s string) id.UserID {
	v, err := id.ParseUserID(s)
	if err != nil {
		panic(err)
	}
	return v
}

var update = flag.Bool("update", false, "update golden files")

func TestGolden_MessageMetadata(t *testing.T) {
	bus := memory.NewMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })

	subscriber := wm.NewSubscriberAdapter(bus)
	_ = wm.NewPublisherAdapter(bus)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	msgCh, err := subscriber.Subscribe(ctx, "order.created")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	evtID := parseEventID("01HK1540X0841Y0A6BSX1VKR96")
	corrID := parseCorrID("01HK1540X0841Y0A6BSX1VKR97")
	causID := parseCausationID("01HK1540X0841Y0A6BSX1VKR98")
	userID := parseUserID("01HK1540X0841Y0A6BSX1VKR99")

	evt, err := event.NewEvent(
		"order.created", aggID, "Order", 1,
		[]byte(`{"item":"widget","qty":3}`),
		event.WithEventID(evtID),
		event.WithOccurredAt(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)),
		event.WithSchemaVersion(2),
		event.WithMetadata(event.Metadata{
			CorrelationID: corrID,
			CausationID:   causID,
			UserID:        userID,
			Source:        "test-service",
			IPAddress:     "10.0.0.1",
			UserAgent:     "test-agent/1.0",
			Custom:        map[event.MetadataKey]string{"custom.trace": "abc123"},
		}),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(ctx, evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case msg := <-msgCh:
		snapshotMetadata(t, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func snapshotMetadata(t *testing.T, msg *message.Message) {
	t.Helper()

	sorted := make(map[string]string)
	for k, v := range msg.Metadata {
		sorted[k] = v
	}

	got, err := json.MarshalIndent(sorted, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "message-metadata.json"),
		got,
		*update,
	)
}
