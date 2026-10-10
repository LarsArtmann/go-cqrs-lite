package deriver

import (
	"context"
	"errors"
	"testing"
	"time"

	cqrscommand "github.com/larsartmann/go-cqrs-lite/command/v4"
	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// asyncFixture wires a dispatcher whose cmd.blocking handler parks until
// released; cmd.unregistered has no handler (Dispatch fails).
type asyncFixture struct {
	dispatcher *cqrscommand.Dispatcher
	entered    chan struct{}
	release    chan struct{}
}

func newAsyncFixture(t *testing.T) *asyncFixture {
	t.Helper()

	f := &asyncFixture{
		dispatcher: cqrscommand.NewDispatcher(),
		entered:    make(chan struct{}),
		release:    make(chan struct{}),
	}

	if err := f.dispatcher.Register("cmd.blocking", cqrscommand.Handler(func(context.Context, cqrscommand.Command) error {
		close(f.entered)

		<-f.release

		return nil
	})); err != nil {
		t.Fatalf("register blocking handler: %v", err)
	}

	return f
}

func blockingDeriver() Deriver {
	return Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmd, _ := cqrscommand.New("cmd.blocking", id.NewStreamID())

		return []cqrscommand.Command{cmd}, nil
	})
}

// TestWithAsyncDispatch_HandlerReturnsBeforeDispatch proves the mode's reason
// to exist: the event handler goroutine is freed immediately (ADR-0154's
// reentrant-publish deadlock cannot engage when dispatch leaves the handler).
func TestWithAsyncDispatch_HandlerReturnsBeforeDispatch(t *testing.T) {
	t.Parallel()

	f := newAsyncFixture(t)
	handler := blockingDeriver().AsHandler(f.dispatcher, WithAsyncDispatch(nil))

	done := make(chan error, 1)
	go func() { done <- handler(context.Background(), testEvent(t, "test.event")) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return while its dispatch was parked — async mode not active")
	}

	select {
	case <-f.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("async dispatch never started")
	}

	close(f.release)
}

// TestWithAsyncDispatch_DefaultIsSynchronous pins the zero-change default:
// without the option, AsHandler still dispatches inside the handler call.
func TestWithAsyncDispatch_DefaultIsSynchronous(t *testing.T) {
	t.Parallel()

	f := newAsyncFixture(t)
	handler := blockingDeriver().AsHandler(f.dispatcher)

	done := make(chan error, 1)
	go func() { done <- handler(context.Background(), testEvent(t, "test.event")) }()

	select {
	case <-f.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("sync dispatch never started")
	}

	// While the dispatch is parked, the handler must NOT have returned.
	select {
	case err := <-done:
		t.Fatalf("handler returned before sync dispatch completed (err=%v)", err)
	default:
	}

	close(f.release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("sync handler never returned after release")
	}
}

// TestWithAsyncDispatch_ErrorSurfacedToCallback proves dispatch failures reach
// the onError callback (with the source event and the failed command) and stop
// that event's remaining commands.
func TestWithAsyncDispatch_ErrorSurfacedToCallback(t *testing.T) {
	t.Parallel()

	f := newAsyncFixture(t)
	secondDispatched := make(chan struct{}, 1)
	if err := f.dispatcher.Register("cmd.second", cqrscommand.Handler(func(context.Context, cqrscommand.Command) error {
		secondDispatched <- struct{}{}

		return nil
	})); err != nil {
		t.Fatalf("register second handler: %v", err)
	}

	d := Deriver(func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		missing, _ := cqrscommand.New("cmd.unregistered", id.NewStreamID())
		second, _ := cqrscommand.New("cmd.second", id.NewStreamID())

		return []cqrscommand.Command{missing, second}, nil
	})

	got := make(chan error, 1)
	handler := d.AsHandler(f.dispatcher, WithAsyncDispatch(func(_ cqrsevent.Event, cmd cqrscommand.Command, err error) {
		if cmd.Type() != "cmd.unregistered" {
			t.Errorf("callback got command %s, want cmd.unregistered", cmd.Type())
		}

		got <- err
	}))

	if err := handler(context.Background(), testEvent(t, "test.event")); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	select {
	case err := <-got:
		if err == nil || errors.Is(err, context.Canceled) {
			t.Fatalf("expected dispatch error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("onError callback never invoked")
	}

	// First-error-stop: the command after the failure must not dispatch.
	select {
	case <-secondDispatched:
		t.Fatal("command after the failed dispatch ran — first-error-stop violated")
	case <-time.After(100 * time.Millisecond):
	}
}

// TestWithAsyncDispatch_ContextCancellationNotInherited pins the
// context.WithoutCancel contract: a cancelled handler context does not
// kill the async dispatch.
func TestWithAsyncDispatch_ContextCancellationNotInherited(t *testing.T) {
	t.Parallel()

	f := newAsyncFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	handler := blockingDeriver().AsHandler(f.dispatcher, WithAsyncDispatch(nil))

	done := make(chan error, 1)
	go func() { done <- handler(ctx, testEvent(t, "test.event")) }()

	<-f.entered
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler blocked on cancelled context — WithoutCancel violated")
	}

	close(f.release)
}
