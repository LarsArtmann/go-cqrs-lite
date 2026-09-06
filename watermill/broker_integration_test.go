package watermill_test

import (
	"bytes"
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// TestRedisStreamRoundtrip verifies the watermill/ bridge (EventBus +
// CommandBus) against a real Redis Streams broker via the official
// watermill-redisstream plugin — the WithBackend contract that ADR-0127
// designates as the canonical broker delivery path.
//
// Usage:
//
//	bash scripts/ephemeral-redis.sh go test -tags "goexperiment.jsonv2" -C watermill -run TestRedis -v ./...
//
// The test is skipped when REDIS_URL is not set, making it safe for CI.
//
// NATS JetStream: no maintained watermill plugin exists (watermill-nats is
// NATS Streaming — deprecated technology built against watermill v1.2-rc).
// Revisit when a JetStream subscriber adapter is available.
func TestRedisStreamRoundtrip(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set — run via: bash scripts/ephemeral-redis.sh go test ...")
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

	pub, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{Client: client}, watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("redis publisher: %v", err)
	}

	evtSub, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{Client: client}, watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("redis event subscriber: %v", err)
	}

	cmdSub, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{Client: client}, watermill.NopLogger{},
	)
	if err != nil {
		t.Fatalf("redis command subscriber: %v", err)
	}

	evtBus := cqrswatermill.NewEventBus(cqrswatermill.WithBackend(pub, evtSub, client))

	t.Cleanup(func() { _ = evtBus.Close() })

	cmdBus := cqrswatermill.NewCommandBus(cqrswatermill.WithCommandBackend(pub, cmdSub, client))
	t.Cleanup(func() { _ = cmdBus.Close() })

	// ── EventBus roundtrip ──────────────────────────────────────────────
	var evtCount atomic.Int32

	var gotEvtType event.Type

	var gotEvtPayload []byte

	err = evtBus.Subscribe("user.created", func(_ context.Context, evt event.Event) error {
		if evtCount.Add(1) == 1 {
			gotEvtType = evt.Type()
			gotEvtPayload = evt.Payload()
		}

		return nil
	})
	if err != nil {
		t.Fatalf("subscribe event: %v", err)
	}

	streamID := id.NewStreamID()
	payload := []byte(`{"name":"alice"}`)

	evt, err := event.NewEvent("user.created", streamID, "User", event.Version(1), payload)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	// Give the broker subscription a moment to attach before publishing.
	time.Sleep(250 * time.Millisecond)

	if err := evtBus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish event: %v", err)
	}

	waitFor(t, func() bool { return evtCount.Load() > 0 }, 5*time.Second)

	if gotEvtType != "user.created" {
		t.Fatalf("event type = %q, want %q", gotEvtType, "user.created")
	}

	if !bytes.Equal(gotEvtPayload, payload) {
		t.Fatalf("event payload = %q, want %q", gotEvtPayload, payload)
	}

	// ── CommandBus roundtrip ────────────────────────────────────────────
	var cmdCount atomic.Int32

	var gotCmdType command.Type

	err = cmdBus.Subscribe("user.create", func(_ context.Context, cmd command.Command) error {
		if cmdCount.Add(1) == 1 {
			gotCmdType = cmd.Type()
		}

		return nil
	})
	if err != nil {
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

	waitFor(t, func() bool { return cmdCount.Load() > 0 }, 5*time.Second)

	if gotCmdType != "user.create" {
		t.Fatalf("command type = %q, want %q", gotCmdType, "user.create")
	}
}
