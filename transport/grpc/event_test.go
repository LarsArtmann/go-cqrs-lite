package grpc_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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

// subscriberCount returns the number of registered handlers. Tests wait on
// it instead of sleeping: publishing before the gRPC stream has subscribed
// the handler onto the bus silently drops the events.
func (b *miniBus) subscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.subs)
}

// waitTimeout bounds condition polling. Generous on purpose: under parallel
// test load, stream establishment regularly exceeds 100ms.
const waitTimeout = 2 * time.Second

// waitFor polls cond until it holds or the deadline expires. On timeout the
// caller's own assertion reports the failure with its full context.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) && !cond() {
		time.Sleep(2 * time.Millisecond)
	}
}

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

// waitSubscribed blocks until the subscriber's gRPC stream has registered a
// handler on the bus, so a subsequent Publish cannot race stream setup.
func (env *eventTestEnv) waitSubscribed(t *testing.T) {
	t.Helper()

	waitFor(t, func() bool { return env.bus.subscriberCount() > 0 })
}

// waitReceived blocks until exactly n event types have been received.
func (env *eventTestEnv) waitReceived(t *testing.T, n int) {
	t.Helper()

	waitFor(t, func() bool {
		env.guard.Lock()
		defer env.guard.Unlock()

		return len(env.received) >= n
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

	env.waitSubscribed(t)

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

	env.waitReceived(t, 2)
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

	env.waitSubscribed(t)

	streamID := id.NewStreamID()

	evt1, _ := event.NewEvent("user.created", streamID, "User", event.Version(1), nil)
	evt2, _ := event.NewEvent("user.deleted", streamID, "User", event.Version(2), nil)

	_ = env.bus.Publish(env.ctx, evt1, evt2)

	env.waitReceived(t, 1)
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

	var gotEvt event.Event

	env.wg.Go(func() {
		_ = env.client.Subscribe(env.ctx, func(_ context.Context, evt event.Event) error {
			env.guard.Lock()
			defer env.guard.Unlock()
			gotEvt = evt

			return nil
		})
	})

	env.waitSubscribed(t)

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

	waitFor(t, func() bool {
		env.guard.Lock()
		defer env.guard.Unlock()

		return gotEvt != nil
	})
	env.stop()

	env.guard.Lock()
	received := gotEvt
	env.guard.Unlock()

	if received == nil {
		t.Fatalf("no event received")
	}

	if received.Encoding() != codec.EncodingCBOR {
		t.Fatalf("received encoding = %q, want cbor", received.Encoding())
	}

	decoded, err := event.DecodePayloadAuto[userCreated](gotEvt)
	if err != nil {
		t.Fatalf("DecodePayloadAuto: %v", err)
	}

	if decoded.Name != "Alice" {
		t.Fatalf("decoded name = %q, want Alice", decoded.Name)
	}
}
