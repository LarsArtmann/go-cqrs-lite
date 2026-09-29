package watermill_test

import (
	"bytes"
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	natsgo "github.com/nats-io/nats.go"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// natsConnCloser adapts *natsgo.Conn to io.Closer (nats.Conn.Close returns
// nothing; Drain drains then closes and reports an error).
type natsConnCloser struct{ conn *natsgo.Conn }

func (c natsConnCloser) Close() error { return c.conn.Drain() }

// TestNatsJetStreamRoundtrip verifies the watermill/ bridge (EventBus +
// CommandBus) against a real NATS JetStream broker via the maintained
// watermill-nats/v2 plugin — the WithBackend contract that ADR-0127
// designates as the canonical broker delivery path (the NATS sibling of
// TestRedisStreamRoundtrip).
//
// Usage:
//
//	bash scripts/ephemeral-nats.sh go test -C watermill -run TestNats -v ./...
//
// The test is skipped when NATS_URL is not set, making it safe for CI.
func TestNatsJetStreamRoundtrip(t *testing.T) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Skip("NATS_URL not set — run via: bash scripts/ephemeral-nats.sh go test ...")
	}

	conn, err := natsgo.Connect(url)
	if err != nil {
		t.Fatalf("nats connect: %v", err)
	}

	t.Cleanup(func() { _ = conn.Drain() })

	pub, err := jetstream.NewPublisher(jetstream.PublisherConfig{
		Conn:           conn,
		TrackMessageID: true,
	})
	if err != nil {
		t.Fatalf("nats publisher: %v", err)
	}

	evtSub, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
		Conn:               conn,
		AckWaitTimeout:     5 * time.Second,
		ResourceInitializer: jetstream.GroupedConsumer("nats-roundtrip-events"),
	})
	if err != nil {
		t.Fatalf("nats event subscriber: %v", err)
	}

	cmdSub, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
		Conn:               conn,
		AckWaitTimeout:     5 * time.Second,
		ResourceInitializer: jetstream.GroupedConsumer("nats-roundtrip-commands"),
	})
	if err != nil {
		t.Fatalf("nats command subscriber: %v", err)
	}

	evtBus := cqrswatermill.NewEventBus(cqrswatermill.WithBackend(pub, evtSub, natsConnCloser{conn}))
	t.Cleanup(func() { _ = evtBus.Close() })

	cmdBus := cqrswatermill.NewCommandBus(cqrswatermill.WithCommandBackend(pub, cmdSub, natsConnCloser{conn}))
	t.Cleanup(func() { _ = cmdBus.Close() })

	// ── EventBus roundtrip ──────────────────────────────────────────────
	var evtCount atomic.Int32

	var gotEvtType event.Type

	var gotEvtPayload []byte

	if err := evtBus.Subscribe("user.created", func(_ context.Context, evt event.Event) error {
		if evtCount.Add(1) == 1 {
			gotEvtType = evt.Type()
			gotEvtPayload = evt.Payload()
		}

		return nil
	}); err != nil {
		t.Fatalf("subscribe event: %v", err)
	}

	streamID := id.NewStreamID()
	payload := []byte(`{"name":"nats-alice"}`)

	evt, err := event.NewEvent("user.created", streamID, "User", event.Version(1), payload)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	// Give the broker subscription a moment to attach before publishing.
	time.Sleep(250 * time.Millisecond)

	if err := evtBus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish event: %v", err)
	}

	waitFor(t, func() bool { return evtCount.Load() > 0 }, 10*time.Second)

	if gotEvtType != "user.created" {
		t.Fatalf("event type = %q, want %q", gotEvtType, "user.created")
	}

	if !bytes.Equal(gotEvtPayload, payload) {
		t.Fatalf("event payload = %q, want %q", gotEvtPayload, payload)
	}

	// ── CommandBus roundtrip ────────────────────────────────────────────
	var cmdCount atomic.Int32

	var gotCmdType command.Type

	if err := cmdBus.Subscribe("user.create", func(_ context.Context, cmd command.Command) error {
		cmdCount.Add(1)
		gotCmdType = cmd.Type()

		return nil
	}); err != nil {
		t.Fatalf("subscribe command: %v", err)
	}

	cmd, err := command.New("user.create", streamID)
	if err != nil {
		t.Fatalf("new command: %v", err)
	}

	time.Sleep(250 * time.Millisecond)

	if err := cmdBus.Publish(context.Background(), cmd); err != nil {
		t.Fatalf("publish command: %v", err)
	}

	waitFor(t, func() bool { return cmdCount.Load() > 0 }, 10*time.Second)

	if gotCmdType != "user.create" {
		t.Fatalf("command type = %q, want %q", gotCmdType, "user.create")
	}
}
