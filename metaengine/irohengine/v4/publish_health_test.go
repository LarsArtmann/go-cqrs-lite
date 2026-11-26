package irohengine_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var errTransportBoom = errors.New("transport boom")

// failingTransport is a Transport whose publish and close outcomes are
// scripted by the test.
type failingTransport struct {
	publishFails atomic.Bool
	closeErr     error
}

func (f *failingTransport) Publish(context.Context, irohengine.WriteOp) error {
	if f.publishFails.Load() {
		return errTransportBoom
	}

	return nil
}

func (*failingTransport) Subscribe(func(irohengine.WriteOp)) error { return nil }

func (f *failingTransport) Close() error { return f.closeErr }

// TestClose_PropagatesTransportError: a transport that fails to shut down is
// reported by Close, not swallowed.
func TestClose_PropagatesTransportError(t *testing.T) {
	t.Parallel()

	tr := &failingTransport{closeErr: errTransportBoom}
	eng := irohengine.Replicated(metaengine.NewMemoryEngine(), irohengine.WithTransport(tr))

	if err := eng.Close(); !errors.Is(err, errTransportBoom) {
		t.Fatalf("Close() = %v, want an error wrapping errTransportBoom", err)
	}
}

// TestHealthCheck_ReportsFailedPublish: a write whose replication failed is
// visible through HealthCheck until a later publish succeeds.
func TestHealthCheck_ReportsFailedPublish(t *testing.T) {
	t.Parallel()

	tr := &failingTransport{}
	eng := irohengine.Replicated(metaengine.NewMemoryEngine(), irohengine.WithTransport(tr))
	t.Cleanup(func() { _ = eng.Close() })

	hc, ok := eng.(metaengine.HealthChecker)
	if !ok {
		t.Fatal("replicated engine does not implement HealthChecker")
	}

	ctx := context.Background()

	tr.publishFails.Store(true)
	if err := eng.(metaengine.MapBackend).MapSet(ctx, "c", "k", "v"); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := hc.HealthCheck(ctx); !errors.Is(err, errTransportBoom) {
		t.Fatalf("HealthCheck after a failed publish = %v, want an error wrapping errTransportBoom", err)
	}

	tr.publishFails.Store(false)
	if err := eng.(metaengine.MapBackend).MapSet(ctx, "c", "k", "v2"); err != nil {
		t.Fatalf("MapSet: %v", err)
	}

	if err := hc.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck after a successful publish = %v, want nil", err)
	}
}
