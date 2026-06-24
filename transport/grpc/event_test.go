package grpc_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3"
)

// miniBus is a minimal event.Bus for testing — fan-out to all subscribers.
type miniBus struct {
	mu   sync.Mutex
	subs []event.Handler
}

func (b *miniBus) Publish(ctx context.Context, events ...event.Event) error {
	b.mu.Lock()
	subs := make([]event.Handler, 0, len(b.subs))
	subs = append(subs, b.subs...)
	b.mu.Unlock()

	for _, evt := range events {
		for _, h := range subs {
			_ = h(ctx, evt)
		}
	}

	return nil
}

func (b *miniBus) Subscribe(_ event.Type, handler event.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subs = append(b.subs, handler)

	return nil
}

func (b *miniBus) SubscribeAll(handler event.Handler) error {
	return b.Subscribe("", handler)
}

func (b *miniBus) Use(...event.Middleware) error {
	return nil
}

func (b *miniBus) UsePublish(...event.PublishMiddleware) error {
	return nil
}

const settleDelay = 100 * time.Millisecond

func TestEventPubSub_RoundTrip(t *testing.T) {
	t.Parallel()

	bus := &miniBus{}

	lis := listen(t)
	srv := grpc.NewServer()

	_, err := cqrsgrpc.RegisterEventService(srv, bus)
	if err != nil {
		t.Fatalf("RegisterEventService: %v", err)
	}

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	client := cqrsgrpc.NewEventClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var guard sync.Mutex

	var received []string

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = client.Subscribe(ctx, func(_ context.Context, evt event.Event) error {
			guard.Lock()
			received = append(received, string(evt.Type()))
			guard.Unlock()

			return nil
		})
	}()

	// Wait for subscription to register, then publish.
	time.Sleep(settleDelay)

	aggID := id.NewAggregateID()

	evt1, _ := event.NewEvent(
		"user.created",
		aggID,
		"User",
		event.Version(1),
		[]byte(`{"name":"Alice"}`),
	)
	evt2, _ := event.NewEvent(
		"user.updated",
		aggID,
		"User",
		event.Version(2),
		[]byte(`{"name":"Bob"}`),
	)

	err = bus.Publish(ctx, evt1, evt2)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	time.Sleep(settleDelay)
	cancel()
	wg.Wait()

	guard.Lock()
	defer guard.Unlock()

	if len(received) != 2 {
		t.Fatalf("received %d events, want 2", len(received))
	}

	if received[0] != "user.created" || received[1] != "user.updated" {
		t.Fatalf("event order: got %v, want [user.created, user.updated]", received)
	}
}

func TestEventPubSub_FilterByType(t *testing.T) {
	t.Parallel()

	bus := &miniBus{}

	lis := listen(t)
	srv := grpc.NewServer()

	_, err := cqrsgrpc.RegisterEventService(srv, bus)
	if err != nil {
		t.Fatalf("RegisterEventService: %v", err)
	}

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	client := cqrsgrpc.NewEventClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var guard sync.Mutex

	var received []string

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = client.Subscribe(ctx, func(_ context.Context, evt event.Event) error {
			guard.Lock()
			received = append(received, string(evt.Type()))
			guard.Unlock()

			return nil
		}, "user.created")
	}()

	time.Sleep(settleDelay)

	aggID := id.NewAggregateID()

	evt1, _ := event.NewEvent("user.created", aggID, "User", event.Version(1), nil)
	evt2, _ := event.NewEvent("user.deleted", aggID, "User", event.Version(2), nil)

	_ = bus.Publish(ctx, evt1, evt2)

	time.Sleep(settleDelay)
	cancel()
	wg.Wait()

	guard.Lock()
	defer guard.Unlock()

	if len(received) != 1 {
		t.Fatalf("received %d events, want 1 (filtered)", len(received))
	}

	if received[0] != "user.created" {
		t.Fatalf("event type: got %s, want user.created", received[0])
	}
}
