package watermill

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
)

func TestSubscriberAdapter_SubscribeAndReceive(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	adapter := NewSubscriberAdapter(bus)
	defer adapter.Close()

	ch, err := adapter.Subscribe(context.Background(), "test.evt")
	if err != nil {
		t.Fatalf("Subscribe(): %v", err)
	}

	evt, _ := eventtest.NewTestEvent()
	go bus.Publish(context.Background(), evt)

	select {
	case msg := <-ch:
		if msg == nil {
			t.Error("received nil message")
		}
	case <-t.Context().Done():
		t.Error("timed out waiting for message")
	}
}

func TestSubscriberAdapter_SubscribeBusError(t *testing.T) {
	t.Parallel()

	bus := &subscribeFailBus{}
	adapter := NewSubscriberAdapter(bus)

	_, err := adapter.Subscribe(context.Background(), "test-topic")
	if err == nil {
		t.Error("Subscribe with failing bus should return error")
	}
}

func TestSubscriberAdapter_CloseIdempotent(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	adapter := NewSubscriberAdapter(bus)

	if err := adapter.Close(); err != nil {
		t.Errorf("first Close(): %v", err)
	}

	if err := adapter.Close(); err != nil {
		t.Errorf("second Close(): %v", err)
	}
}

type subscribeFailBus struct{}

func (s *subscribeFailBus) Subscribe(event.Type, event.Handler) error {
	return errors.New("subscribe failed")
}
func (s *subscribeFailBus) SubscribeAll(event.Handler) error              { return nil }
func (s *subscribeFailBus) Publish(context.Context, ...event.Event) error { return nil }
func (s *subscribeFailBus) Close() error                                  { return nil }
func (s *subscribeFailBus) Use(...event.Middleware) error                 { return nil }
func (s *subscribeFailBus) UsePublish(...event.PublishMiddleware) error   { return nil }
