package watermill_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
	natsgo "github.com/nats-io/nats.go"
	natsjs "github.com/nats-io/nats.go/jetstream"
)

// natsConnCloser adapts *natsgo.Conn to io.Closer (nats.Conn.Close returns
// nothing; Drain drains then closes and reports an error).
type natsConnCloser struct{ conn *natsgo.Conn }

func (c natsConnCloser) Close() error { return c.conn.Drain() }

// natsStreamName sanitizes a topic into a legal JetStream stream name: NATS
// stream names reject '.', '*', '>', spaces and tabs, but SUBJECTS want the
// dotted form. The watermill-nats/v2 built-in initializers derive the stream
// name from the topic VERBATIM (upstream gap, v2.2.0) — dotted event types
// like "user.created" need this custom mapping on both sides.
func natsStreamName(topic string) string {
	return strings.NewReplacer(".", "_", "*", "_", ">", "_", " ", "_", "\t", "_").Replace(topic)
}

// bridgeStreamConfig maps a (possibly dotted) topic to a WorkQueue stream
// under a sanitized name. The plugin routes by stream NAME as the SUBJECT
// too (the marshaler receives streamConfig.Name), so the sanitized name is
// what the subject must be — the dotted topic never reaches the wire.
func bridgeStreamConfig(topic string) natsjs.StreamConfig {
	name := natsStreamName(topic)

	return natsjs.StreamConfig{
		Name:      name,
		Subjects:  []string{name},
		Retention: natsjs.WorkQueuePolicy,
	}
}

// bridgeConsumerInitializer provisions (or attaches to) the stream and
// consumer for a dotted topic — the ResourceInitializer the plugin's
// built-ins cannot express (they derive stream names verbatim from topics).
func bridgeConsumerInitializer(
	ctx context.Context, js natsjs.JetStream, topic string,
) (natsjs.Consumer, func(context.Context, watermill.LoggerAdapter), error) {
	stream, err := js.CreateOrUpdateStream(ctx, bridgeStreamConfig(topic))
	if err != nil {
		return nil, nil, fmt.Errorf("create stream for topic %s: %w", topic, err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, natsjs.ConsumerConfig{
		Name:      "watermill__" + natsStreamName(topic),
		AckPolicy: natsjs.AckExplicitPolicy,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create consumer for topic %s: %w", topic, err)
	}

	return consumer, nil, nil
}

// ensureBridgeStream provisions the stream for a bus topic via the shared
// sanitized-name mapping (see bridgeStreamConfig).
func ensureBridgeStream(t *testing.T, conn *natsgo.Conn, topic string) {
	t.Helper()

	js, err := natsjs.New(conn)
	if err != nil {
		t.Fatalf("jetstream handle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(ctx, bridgeStreamConfig(topic)); err != nil {
		t.Fatalf("create stream for %s: %v", topic, err)
	}
}

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

	// The bridge publishes everything to fixed topics (DefaultEventBusTopic
	// "cqrs.events" / DefaultCommandBusTopic "cqrs.commands") — both dotted,
	// so the bus streams are pre-created here under sanitized names (the
	// plugin derives stream names verbatim from topics — upstream gap).
	ensureBridgeStream(t, conn, cqrswatermill.DefaultEventBusTopic)
	ensureBridgeStream(t, conn, cqrswatermill.DefaultCommandBusTopic)

	pub, err := jetstream.NewPublisher(jetstream.PublisherConfig{
		Conn:            conn,
		TrackMessageID:  true,
		ConfigureStream: bridgeStreamConfig,
	})
	if err != nil {
		t.Fatalf("nats publisher: %v", err)
	}

	evtSub, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
		Conn:                conn,
		AckWaitTimeout:      5 * time.Second,
		ResourceInitializer: bridgeConsumerInitializer,
	})
	if err != nil {
		t.Fatalf("nats event subscriber: %v", err)
	}

	cmdSub, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
		Conn:                conn,
		AckWaitTimeout:      5 * time.Second,
		ResourceInitializer: bridgeConsumerInitializer,
	})
	if err != nil {
		t.Fatalf("nats command subscriber: %v", err)
	}

	evtBus := cqrswatermill.NewEventBus(
		cqrswatermill.WithBackend(pub, evtSub, natsConnCloser{conn}),
	)
	t.Cleanup(func() { _ = evtBus.Close() })

	cmdBus := cqrswatermill.NewCommandBus(
		cqrswatermill.WithCommandBackend(pub, cmdSub, natsConnCloser{conn}),
	)
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
