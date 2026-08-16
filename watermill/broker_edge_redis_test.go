package watermill_test

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

// Broker-edge tests against a real Redis Streams broker — the edges the
// in-process gochannel backend structurally cannot catch: Nack redelivery,
// consumer-group exactly-once delivery, and large-payload integrity.
//
// Usage: nix run .#integration-redis
// (or: bash scripts/ephemeral-redis.sh go test ./...)

// newRedisEdgeClient returns a pinged client, skipping when REDIS_URL is unset.
func newRedisEdgeClient(t *testing.T) redis.UniversalClient {
	t.Helper()

	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set — run via: nix run .#integration-redis")
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse REDIS_URL: %v", err)
	}

	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}

	return client
}

// receiveOne reads one message from ch within timeout and returns it (nil on
// timeout), so tests can assert presence/absence without deadlocking.
func receiveOne(ch <-chan *message.Message, timeout time.Duration) *message.Message {
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		return nil
	}
}

func TestRedisStream_NackRedelivers(t *testing.T) {
	client := newRedisEdgeClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const topic = "edge-redeliver"

	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:          client,
		ConsumerGroup:   "edge-redeliver-group",
		Consumer:        "edge-redeliver-c1",
		NackResendSleep: 200 * time.Millisecond,
	}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("subscriber: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	msgs, err := sub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	pub, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{Client: client}, watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}

	time.Sleep(250 * time.Millisecond) // let the subscription attach

	if err := pub.Publish(
		topic,
		message.NewMessage(watermill.NewUUID(), []byte(`{"edge":"nack"}`)),
	); err != nil {
		t.Fatalf("publish: %v", err)
	}

	first := receiveOne(msgs, 5*time.Second)
	if first == nil {
		t.Fatal("first delivery never arrived")
	}
	first.Nack()

	second := receiveOne(msgs, 5*time.Second)
	if second == nil {
		t.Fatal("nacked message was never redelivered")
	}
	second.Ack()
}

func TestRedisStream_ConsumerGroupExactlyOnce(t *testing.T) {
	client := newRedisEdgeClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const (
		topic = "edge-rebalance"
		group = "edge-rebalance-group"
		total = 20
	)

	mkSub := func(name string) <-chan *message.Message {
		t.Helper()

		sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
			Client:        client,
			ConsumerGroup: group,
			Consumer:      name,
		}, watermill.NopLogger{})
		if err != nil {
			t.Fatalf("subscriber %s: %v", name, err)
		}
		t.Cleanup(func() { _ = sub.Close() })

		ch, err := sub.Subscribe(ctx, topic)
		if err != nil {
			t.Fatalf("subscribe %s: %v", name, err)
		}

		return ch
	}

	chA, chB := mkSub("edge-rebalance-a"), mkSub("edge-rebalance-b")

	pub, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{Client: client}, watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}

	time.Sleep(250 * time.Millisecond)

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
	}, 15*time.Second)

	mu.Lock()
	defer mu.Unlock()

	if len(seen) != total {
		t.Fatalf("consumer group delivered %d/%d unique messages (duplicates or losses)",
			len(seen), total)
	}
}

func TestRedisStream_LargePayloadRoundtrip(t *testing.T) {
	client := newRedisEdgeClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const topic = "edge-large"

	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        client,
		ConsumerGroup: "edge-large-group",
		Consumer:      "edge-large-c1",
	}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("subscriber: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	msgs, err := sub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	pub, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{Client: client}, watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}

	time.Sleep(250 * time.Millisecond)

	payload := bytes.Repeat([]byte("0123456789abcdef"), 128*1024) // 2 MiB
	if len(payload) != 2*1024*1024 {
		t.Fatalf("payload = %d bytes, want %d", len(payload), 2*1024*1024)
	}

	if err := pub.Publish(topic, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		t.Fatalf("publish large payload: %v", err)
	}

	msg := receiveOne(msgs, 10*time.Second)
	if msg == nil {
		t.Fatal("large payload never arrived")
	}
	msg.Ack()

	if !bytes.Equal(msg.Payload, payload) {
		t.Fatalf("payload corrupted: got %d bytes, want %d", len(msg.Payload), len(payload))
	}
}
