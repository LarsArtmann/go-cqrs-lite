package irohengine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	irohengine "github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestHealthCheck_LocalOnlyEngine(t *testing.T) {
	t.Parallel()

	eng := irohengine.Replicated(metaengine.NewMemoryEngine())
	t.Cleanup(func() { _ = eng.Close() })

	hc, ok := eng.(metaengine.HealthChecker)
	if !ok {
		t.Fatal("Replicated engine must implement metaengine.HealthChecker")
	}

	if err := hc.HealthCheck(context.Background()); err != nil {
		t.Fatalf("healthy local-only engine reported unhealthy: %v", err)
	}
}

func TestHealthCheck_HealthyNetwork(t *testing.T) {
	t.Parallel()

	net := irohengine.NewInProcessNetwork()
	eng := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithTransport(net.Join("a")),
	)
	t.Cleanup(func() { _ = eng.Close() })

	hc, ok := eng.(metaengine.HealthChecker)
	if !ok {
		t.Fatal("Replicated engine must implement metaengine.HealthChecker")
	}

	if err := hc.HealthCheck(context.Background()); err != nil {
		t.Fatalf("engine on open network reported unhealthy: %v", err)
	}
}

func TestHealthCheck_ClosedNetworkIsUnhealthy(t *testing.T) {
	t.Parallel()

	net := irohengine.NewInProcessNetwork()
	eng := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithTransport(net.Join("a")),
	)

	hc, ok := eng.(metaengine.HealthChecker)
	if !ok {
		t.Fatal("Replicated engine must implement metaengine.HealthChecker")
	}

	net.Shutdown()

	err := hc.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("closed network must surface as an unhealthy engine")
	}

	if !errors.Is(err, irohengine.ErrTransportClosed) {
		t.Fatalf("want ErrTransportClosed in chain, got: %v", err)
	}

	if !strings.Contains(err.Error(), "transport") {
		t.Errorf("error should attribute the failure to the transport: %v", err)
	}
}

func TestInProcessNetwork_HealthyAfterClose(t *testing.T) {
	t.Parallel()

	net := irohengine.NewInProcessNetwork()

	if err := net.Healthy(context.Background()); err != nil {
		t.Fatalf("open network reported unhealthy: %v", err)
	}

	net.Shutdown()

	if err := net.Healthy(context.Background()); !errors.Is(err, irohengine.ErrTransportClosed) {
		t.Fatalf("shutdown network Healthy = %v, want ErrTransportClosed", err)
	}
}

func TestHealthCheck_ClosedPeerTransportIsUnhealthy(t *testing.T) {
	t.Parallel()

	net := irohengine.NewInProcessNetwork()
	peer := net.Join("a")
	eng := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithTransport(peer),
	)
	t.Cleanup(func() { _ = eng.Close() })

	hc, ok := eng.(metaengine.HealthChecker)
	if !ok {
		t.Fatal("Replicated engine must implement metaengine.HealthChecker")
	}

	if err := peer.Close(); err != nil {
		t.Fatalf("peer Close: %v", err)
	}

	err := hc.HealthCheck(context.Background())
	if !errors.Is(err, irohengine.ErrTransportClosed) {
		t.Fatalf("closed peer transport must surface as unhealthy, got: %v", err)
	}
}
