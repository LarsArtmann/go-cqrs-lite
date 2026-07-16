package watermill_test

import (
	"context"
	"encoding/json/v2"
	"flag"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	wm "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_MessageMetadata(t *testing.T) {
	bus := eventtest.NewFakeBus()
	t.Cleanup(func() { _ = bus.Close() })

	subscriber := wm.NewSubscriberAdapter(bus)
	_ = wm.NewPublisherAdapter(bus)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	msgCh, err := subscriber.Subscribe(ctx, "order.created")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	aggID := idtest.ParseAggregateID(t, "01HK1540X0841Y0A6BSX1VKR95")
	evtID := idtest.ParseEventID(t, "01HK1540X0841Y0A6BSX1VKR96")
	corrID := idtest.ParseCorrelationID(t, "01HK1540X0841Y0A6BSX1VKR97")
	causID := idtest.ParseCausationID(t, "01HK1540X0841Y0A6BSX1VKR98")
	userID := idtest.ParseUserID(t, "01HK1540X0841Y0A6BSX1VKR99")

	evt, err := event.NewEvent(
		"order.created", aggID, "Order", 1,
		[]byte(`{"item":"widget","qty":3}`),
		event.WithEventID(evtID),
		event.WithOccurredAt(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)),
		event.WithSchemaVersion(2),
		event.WithMetadata(event.Metadata{
			Tracing: metadata.Tracing{
				CorrelationID: corrID,
				CausationID:   causID,
				UserID:        userID,
			},
			Source:    "test-service",
			IPAddress: "10.0.0.1",
			UserAgent: "test-agent/1.0",
			Custom:    map[event.MetadataKey]string{"custom.trace": "abc123"},
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

	got := marshalSortedMap(sorted)

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "message-metadata.json"),
		got,
		*update,
	)
}

func marshalSortedMap(m map[string]string) []byte {
	keys := slices.Sorted(maps.Keys(m))
	var b strings.Builder
	b.WriteString("{\n")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",\n")
		}
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(m[k])
		b.WriteString("  ")
		b.Write(kb)
		b.WriteString(": ")
		b.Write(vb)
	}
	b.WriteString("\n}")

	return []byte(b.String())
}
