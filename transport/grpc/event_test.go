package grpc_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"
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

// eventTestEnv bundles the shared gRPC server, client, and subscriber state
// used by the event pub/sub tests.
type eventTestEnv struct {
	bus      *miniBus
	client   *cqrsgrpc.EventClient
	ctx      context.Context //nolint:containedctx // test helper, scoped to test lifecycle
	cancel   context.CancelFunc
	guard    sync.Mutex
	received []string
	wg       sync.WaitGroup
}

// newEventTestEnv starts a gRPC server with an event service, connects a
// client, and returns the wired environment. Cleanup is registered via t.
func newEventTestEnv(t *testing.T) *eventTestEnv {
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

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return &eventTestEnv{
		bus:    bus,
		client: cqrsgrpc.NewEventClient(conn),
		ctx:    ctx,
		cancel: cancel,
	}
}

// subscribe starts a background goroutine that collects event types into
// env.received. filters are optional event-type filters forwarded to Subscribe.
func (env *eventTestEnv) subscribe(filters ...string) {
	env.wg.Go(func() {
		_ = env.client.Subscribe(env.ctx, func(_ context.Context, evt event.Event) error {
			env.guard.Lock()
			env.received = append(env.received, string(evt.Type()))
			env.guard.Unlock()

			return nil
		}, filters...)
	})
}

// stop cancels the context and waits for the subscriber goroutine to exit.
func (env *eventTestEnv) stop() {
	env.cancel()
	env.wg.Wait()
}

func TestEventPubSub_RoundTrip(t *testing.T) {
	env := newEventTestEnv(t)

	env.subscribe()

	// Wait for subscription to register, then publish.
	time.Sleep(settleDelay)

	streamID := id.NewStreamID()

	evt1, _ := event.NewEvent(
		"user.created",
		streamID,
		"User",
		event.Version(1),
		[]byte(`{"name":"Alice"}`),
	)
	evt2, _ := event.NewEvent(
		"user.updated",
		streamID,
		"User",
		event.Version(2),
		[]byte(`{"name":"Bob"}`),
	)

	err := env.bus.Publish(env.ctx, evt1, evt2)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	time.Sleep(settleDelay)
	env.stop()

	env.guard.Lock()
	defer env.guard.Unlock()

	if len(env.received) != 2 {
		t.Fatalf("received %d events, want 2", len(env.received))
	}

	if env.received[0] != "user.created" || env.received[1] != "user.updated" {
		t.Fatalf("event order: got %v, want [user.created, user.updated]", env.received)
	}
}

func TestEventPubSub_FilterByType(t *testing.T) {
	env := newEventTestEnv(t)

	env.subscribe("user.created")

	// Wait for subscription to register, then publish.
	time.Sleep(settleDelay)

	streamID := id.NewStreamID()

	evt1, _ := event.NewEvent("user.created", streamID, "User", event.Version(1), nil)
	evt2, _ := event.NewEvent("user.deleted", streamID, "User", event.Version(2), nil)

	_ = env.bus.Publish(env.ctx, evt1, evt2)

	time.Sleep(settleDelay)
	env.stop()

	env.guard.Lock()
	defer env.guard.Unlock()

	if len(env.received) != 1 {
		t.Fatalf("received %d events, want 1 (filtered)", len(env.received))
	}

	if env.received[0] != "user.created" {
		t.Fatalf("event type: got %s, want user.created", env.received[0])
	}
}

func TestEventPubSub_PreservesCBOREncoding(t *testing.T) {
	env := newEventTestEnv(t)

	type userCreated struct {
		Name string `json:"name"`
	}

	var (
		gotEvt  event.Event
		gotOnce sync.Once
	)

	env.wg.Go(func() {
		_ = env.client.Subscribe(env.ctx, func(_ context.Context, evt event.Event) error {
			gotOnce.Do(func() { gotEvt = evt })

			return nil
		})
	})

	time.Sleep(settleDelay)

	streamID := id.NewStreamID()

	evt, err := event.New(
		"user.created", streamID, "User", event.Version(1),
		userCreated{Name: "Alice"},
		event.WithCodec(codec.CBORCodec{}),
	)
	if err != nil {
		t.Fatalf("create CBOR event: %v", err)
	}

	if evt.Encoding() != codec.EncodingCBOR {
		t.Fatalf("source encoding = %q, want cbor", evt.Encoding())
	}

	if err := env.bus.Publish(env.ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	time.Sleep(settleDelay)
	env.stop()

	if gotEvt == nil {
		t.Fatalf("no event received")
	}

	if gotEvt.Encoding() != codec.EncodingCBOR {
		t.Fatalf("received encoding = %q, want cbor", gotEvt.Encoding())
	}

	decoded, err := event.DecodePayloadAuto[userCreated](gotEvt)
	if err != nil {
		t.Fatalf("DecodePayloadAuto: %v", err)
	}

	if decoded.Name != "Alice" {
		t.Fatalf("decoded name = %q, want Alice", decoded.Name)
	}
}
