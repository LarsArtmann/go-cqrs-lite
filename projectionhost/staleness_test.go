package projectionhost_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

func newStalenessHost(t *testing.T) (*projectionhost.Host, *memoryJournal) {
	t.Helper()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	host, err := projectionhost.New(journal, cpStore)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_ = host.Register(&countingProjection{name: "test", eventTypes: nil})

	return host, journal
}

func TestCheckStaleness_DisabledByZero(t *testing.T) {
	t.Parallel()

	host, _ := newStalenessHost(t)

	if err := host.CheckStaleness(0); err != nil {
		t.Fatalf("CheckStaleness(0): got %v, want nil", err)
	}
}

func TestCheckStaleness_DisabledByNegative(t *testing.T) {
	t.Parallel()

	host, _ := newStalenessHost(t)

	if err := host.CheckStaleness(-1 * time.Second); err != nil {
		t.Fatalf("CheckStaleness(-1s): got %v, want nil", err)
	}
}

func TestCheckStaleness_FreshProjection(t *testing.T) {
	host, journal := newStalenessHost(t)
	journal.append(makeEvent("test.event"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = host.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	host.Stop()

	// Just processed — generous threshold should pass.
	if err := host.CheckStaleness(10 * time.Second); err != nil {
		t.Fatalf("CheckStaleness(10s): got %v, want nil", err)
	}
}

func TestCheckStaleness_StaleProjection(t *testing.T) {
	host, journal := newStalenessHost(t)
	journal.append(makeEvent("test.event"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = host.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	host.Stop()

	// Wait so the lag exceeds the tiny threshold.
	time.Sleep(50 * time.Millisecond)

	err := host.CheckStaleness(1 * time.Millisecond)
	if err == nil {
		t.Fatal("CheckStaleness(1ms): expected error, got nil")
	}

	if !errors.Is(err, projectionhost.ErrProjectionStale) {
		t.Fatalf("expected ErrProjectionStale, got %v", err)
	}
}

func TestCheckStaleness_NoEventsProcessed(t *testing.T) {
	host, _ := newStalenessHost(t)

	// No events processed (lag == 0) → considered fresh.
	if err := host.CheckStaleness(1 * time.Millisecond); err != nil {
		t.Fatalf("CheckStaleness with no events: got %v, want nil", err)
	}
}

func TestCheckProjectionStaleness_UnknownProjection(t *testing.T) {
	host, _ := newStalenessHost(t)

	err := host.CheckProjectionStaleness("nonexistent", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for unknown projection, got nil")
	}
}

func TestCheckProjectionStaleness_Fresh(t *testing.T) {
	host, journal := newStalenessHost(t)
	journal.append(makeEvent("test.event"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = host.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	host.Stop()

	if err := host.CheckProjectionStaleness("test", 10*time.Second); err != nil {
		t.Fatalf("CheckProjectionStaleness: got %v, want nil", err)
	}
}

func TestCheckProjectionStaleness_Stale(t *testing.T) {
	host, journal := newStalenessHost(t)
	journal.append(makeEvent("test.event"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = host.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	host.Stop()

	time.Sleep(50 * time.Millisecond)

	err := host.CheckProjectionStaleness("test", 1*time.Millisecond)
	if !errors.Is(err, projectionhost.ErrProjectionStale) {
		t.Fatalf("expected ErrProjectionStale, got %v", err)
	}
}
