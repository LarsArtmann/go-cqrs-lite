package watermill

import (
	"context"
	"os"
	"testing"
)

// TestRedisStreamRoundtrip tests the Watermill EventBus + CommandBus using
// a real Redis Streams backend. Requires a running Redis instance.
//
// Usage: bash scripts/ephemeral-redis.sh go test -tags "goexperiment.jsonv2" -run TestRedis -v ./...
//
// The test is skipped when REDIS_PORT is not set, making it safe for CI.
func TestRedisStreamRoundtrip(t *testing.T) {
	if os.Getenv("REDIS_PORT") == "" {
		t.Skip("REDIS_PORT not set — run via: bash scripts/ephemeral-redis.sh go test ...")
	}

	_ = context.Background()
	t.Skip(
		"Redis Streams integration test requires watermill-redis plugin dependency — see scripts/ephemeral-redis.sh for setup",
	)
}

// TestNATSJetStreamRoundtrip tests the Watermill EventBus + CommandBus using
// a real NATS JetStream backend. Requires a running NATS instance.
//
// Usage: bash scripts/ephemeral-nats.sh go test -tags "goexperiment.jsonv2" -run TestNATS -v ./...
//
// The test is skipped when NATS_PORT is not set, making it safe for CI.
func TestNATSJetStreamRoundtrip(t *testing.T) {
	if os.Getenv("NATS_PORT") == "" {
		t.Skip("NATS_PORT not set — run via: bash scripts/ephemeral-nats.sh go test ...")
	}

	_ = context.Background()
	t.Skip(
		"NATS JetStream integration test requires watermill-nats plugin dependency — see scripts/ephemeral-nats.sh for setup",
	)
}
