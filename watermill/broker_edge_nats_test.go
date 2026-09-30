package watermill_test

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/ThreeDotsLabs/watermill/message"
	natsgo "github.com/nats-io/nats.go"
	natsjs "github.com/nats-io/nats.go/jetstream"
)

// NATS JetStream broker-edge suite — the sibling of the Redis Streams edge
// suite (broker_edge_redis_test.go): Nack redelivery, group exactly-once
// (TrackMsgID + grouped durable consumers), and 2 MiB payload roundtrip
// (requires the ephemeral server's raised --max_payload).
//
// Run via: bash scripts/ephemeral-nats.sh go test -C watermill -run TestNats -v ./...

func newNatsEdgeConn(t *testing.T) *natsgo.Conn {
	t.Helper()

	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Skip("NATS_URL not set — run via: bash scripts/ephemeral-nats.sh go test ...")
	}

	conn, err := natsgo.Connect(url)
	if err != nil {
		t.Fatalf("nats connect: %v", err)
	}

	t.Cleanup(func() { _ = conn.Drain() })

	return conn
}

// ensureNatsStream pre-creates the topic's stream (WorkQueue retention, the
// GroupedConsumer configurator's shape): the plugin's subscribers only GET
// streams — they never provision — so a subscribe-before-publish needs the
// stream to exist.
func ensureNatsStream(t *testing.T, conn *natsgo.Conn, topic string) {
	t.Helper()

	js, err := natsjs.New(conn)
	if err != nil {
		t.Fatalf("jetstream handle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(ctx, natsjs.StreamConfig{
		Name:      topic,
		Subjects:  []string{topic},
		Retention: natsjs.WorkQueuePolicy,
	}); err != nil {
		t.Fatalf("create stream %s: %v", topic, err)
	}
}

// TestNatsJetStream_NackRedelivers pins that a Nacked message comes back:
// JetStream terminally NAKs nothing until the consumer's max deliver lapses —
// the same at-least-once contract the Redis Streams leg pins.
func TestNatsJetStream_NackRedelivers(t *testing.T) {
	const topic = "nats-edge-redeliver"

	conn := newNatsEdgeConn(t)
	ensureNatsStream(t, conn, topic)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	sub, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
		Conn:                conn,
		AckWaitTimeout:      5 * time.Second,
		ResourceInitializer: jetstream.GroupedConsumer("nats-edge-redeliver"),
	})
	if err != nil {
		t.Fatalf("subscriber: %v", err)
	}

	t.Cleanup(func() { _ = sub.Close() })

	msgs, err := sub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	pub, err := jetstream.NewPublisher(jetstream.PublisherConfig{Conn: conn})
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}

	time.Sleep(500 * time.Millisecond) // let the subscription + stream attach

	if err := pub.Publish(
		topic,
		message.NewMessage(watermill.NewUUID(), []byte(`{"edge":"nack"}`)),
	); err != nil {
		t.Fatalf("publish: %v", err)
	}

	first := receiveOne(msgs, 10*time.Second)
	if first == nil {
		t.Fatal("first delivery never arrived")
	}

	first.Nack()

	second := receiveOne(msgs, 10*time.Second)
	if second == nil {
		t.Fatal("nacked message was never redelivered")
	}

	second.Ack()
}

// TestNatsJetStream_ConsumerGroupExactlyOnce pins the grouped-consumer
// fan-out + TrackMsgID dedup recipe: two subscribers sharing one group
// receive each of the published messages exactly once between them.
func TestNatsJetStream_ConsumerGroupExactlyOnce(t *testing.T) {
	conn := newNatsEdgeConn(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const (
		topic = "nats-edge-group"
		group = "nats-edge-group"
		total = 20
	)

	ensureNatsStream(t, conn, topic)

	mkSub := func() <-chan *message.Message {
		t.Helper()

		sub, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
			Conn:                conn,
			AckWaitTimeout:      5 * time.Second,
			ResourceInitializer: jetstream.GroupedConsumer(group),
		})
		if err != nil {
			t.Fatalf("subscriber: %v", err)
		}

		t.Cleanup(func() { _ = sub.Close() })

		ch, err := sub.Subscribe(ctx, topic)
		if err != nil {
			t.Fatalf("subscribe: %v", err)
		}

		return ch
	}

	chA, chB := mkSub(), mkSub()

	// TrackMessageID maps the watermill UUID to the NATS MsgId — the
	// server-side exactly-once dedup (plugin docs recipe).
	pub, err := jetstream.NewPublisher(jetstream.PublisherConfig{
		Conn:           conn,
		TrackMessageID: true,
	})
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	for i := range total {
		if err := pub.Publish(
			topic,
			message.NewMessage(watermill.NewUUID(), []byte(`{"i":`+string(rune('0'+i))+`}`)),
		); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	var mu sync.Mutex

	seen := make(map[string]struct{})

	collect := func(ch <-chan *message.Message) {
		for msg := range ch {
			msg.Ack()

			mu.Lock()
			seen[msg.UUID] = struct{}{}
			mu.Unlock()
		}
	}

	go collect(chA)
	go collect(chB)

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return len(seen) == total
	}, 20*time.Second)

	mu.Lock()
	defer mu.Unlock()

	if len(seen) != total {
		t.Fatalf("consumer group delivered %d/%d unique messages (duplicates or losses)",
			len(seen), total)
	}
}

// TestNatsJetStream_LargePayloadRoundtrip pins the 2 MiB payload class
// through JetStream persistence (server started with --max_payload 8MB; the
// NATS default 1MB cap would reject the publish).
func TestNatsJetStream_LargePayloadRoundtrip(t *testing.T) {
	const topic = "nats-edge-large"

	conn := newNatsEdgeConn(t)
	ensureNatsStream(t, conn, topic)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	sub, err := jetstream.NewSubscriber(jetstream.SubscriberConfig{
		Conn:                conn,
		AckWaitTimeout:      10 * time.Second,
		ResourceInitializer: jetstream.GroupedConsumer("nats-edge-large"),
	})
	if err != nil {
		t.Fatalf("subscriber: %v", err)
	}

	t.Cleanup(func() { _ = sub.Close() })

	msgs, err := sub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	pub, err := jetstream.NewPublisher(jetstream.PublisherConfig{Conn: conn})
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	payload := bytes.Repeat([]byte("0123456789abcdef"), 128*1024) // 2 MiB
	if len(payload) != 2*1024*1024 {
		t.Fatalf("payload = %d bytes, want %d", len(payload), 2*1024*1024)
	}

	if err := pub.Publish(topic, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		t.Fatalf("publish large payload: %v", err)
	}

	msg := receiveOne(msgs, 15*time.Second)
	if msg == nil {
		t.Fatal("large payload never arrived")
	}

	msg.Ack()

	if !bytes.Equal(msg.Payload, payload) {
		t.Fatalf("payload corrupted: got %d bytes, want %d", len(msg.Payload), len(payload))
	}
}
