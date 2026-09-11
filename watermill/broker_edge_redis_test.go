package watermill_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	cqrs "github.com/larsartmann/go-cqrs-lite/watermill/v4"
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

// TestRedisStream_CatchUpReplayThroughput runs the CatchUpSubscriber replay
// path with a REAL Redis Streams broker as the live side — the broker-backed
// variant of BenchmarkCatchUp_ReplayThroughput (in-memory gochannel). It
// pins count, ordering, and the replay→live handoff end-to-end, and LOGS the
// observed throughput without asserting a ceiling (shared runners vary).
// Requires REDIS_URL (nix run .#integration-redis).
func TestRedisStream_CatchUpReplayThroughput(t *testing.T) {
	client := newRedisEdgeClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const (
		topic       = "edge-catchup-throughput"
		replayCount = 1000
		liveCount   = 10
	)

	// Journal side: replayCount sequence-tagged events from the FakeStore.
	store := eventtest.NewFakeStore()
	streamID := id.NewStreamID()

	events := make([]event.Event, 0, replayCount)
	for i := range replayCount {
		evt, evtErr := event.NewEvent(
			topic, streamID, "TestStream", event.Version(i+1),
			[]byte(fmt.Sprintf(`{"n":%d}`, i)),
		)
		if evtErr != nil {
			t.Fatalf("NewEvent(%d): %v", i, evtErr)
		}
		events = append(events, evt)
	}

	if err := store.AppendBatch(ctx,
		id.NewStreamRef("TestStream", streamID), events); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	// Live side: a real Redis Streams subscription (the whole point — the
	// in-memory gochannel bus cannot catch broker-edge behavior).
	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        client,
		ConsumerGroup: "edge-catchup-group",
		Consumer:      "edge-catchup-c1",
	}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("subscriber: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	catchUp, err := cqrs.NewCatchUpSubscriber(
		store, cqrs.NewSubscriberAdapter(sub),
		memory.NewMemoryCheckpointStore(), nil,
	)
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}
	t.Cleanup(func() { _ = catchUp.Close() })

	ch, err := catchUp.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	deadline := time.After(60 * time.Second)

	// Phase 1: consume+ack the whole replay in order, timing it.
	start := time.Now()

	for i := range replayCount {
		var msg *message.Message

		select {
		case msg = <-ch:
		case <-deadline:
			t.Fatalf("timed out at replayed message %d", i)
		}

		if got := msg.Metadata.Get("event_id"); got != events[i].ID().String() {
			t.Fatalf("replay order broken at %d: event_id=%s", i, got)
		}
		msg.Ack()
	}
	elapsed := time.Since(start)
	t.Logf("replay throughput: %d events in %s (%.0f events/sec)",
		replayCount, elapsed.Round(time.Millisecond), float64(replayCount)/elapsed.Seconds())

	// Phase 2: live messages through the real broker, after replay drained —
	// pins the replay→live handoff over Redis Streams.
	pub, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{Client: client}, watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}

	for i := range liveCount {
		payload := fmt.Appendf(nil, `{"live":%d}`, i)
		if err := pub.Publish(topic, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
			t.Fatalf("publish live %d: %v", i, err)
		}
	}

	for i := range liveCount {
		var msg *message.Message

		select {
		case msg = <-ch:
		case <-deadline:
			t.Fatalf("timed out waiting for live message %d", i)
		}

		msg.Ack()
	}
}
