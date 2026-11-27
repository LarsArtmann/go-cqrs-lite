package watermill

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// publishManyAfterClose is the regression driver for the send-on-closed-
// channel panic: the handlers stay registered on the bus after Close, so
// every publish below re-enters a handler whose output channel is closed.
// A closed send case and a closed closeCh case are both ready, so a single
// publish may pass by luck; hundreds make a panic virtually certain.
const publishesAfterClose = 200

func publishEventsAfterClose(t *testing.T, bus *eventtest.FakeBus) {
	t.Helper()

	for range publishesAfterClose {
		evt, err := eventtest.NewTestEvent()
		if err != nil {
			t.Fatalf("NewTestEvent(): %v", err)
		}

		_ = bus.Publish(context.Background(), evt)
	}
}

func TestSubscriberAdapter_PublishAfterCloseDoesNotPanic(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	adapter := NewSubscriberAdapter(bus)

	if _, err := adapter.Subscribe(context.Background(), "test.evt"); err != nil {
		t.Fatalf("Subscribe(): %v", err)
	}

	if err := adapter.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	publishEventsAfterClose(t, bus)
}

func TestSubscriberAdapter_CloseDrainsOutputChannel(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	adapter := NewSubscriberAdapter(bus)

	ch, err := adapter.Subscribe(context.Background(), "test.evt")
	if err != nil {
		t.Fatalf("Subscribe(): %v", err)
	}

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("NewTestEvent(): %v", err)
	}

	_ = bus.Publish(context.Background(), evt)

	if err := adapter.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	select {
	case msg, ok := <-ch:
		if !ok || msg == nil {
			t.Fatal("buffered message must stay readable after Close")
		}
	default:
		t.Fatal("buffered message lost on Close")
	}

	if _, ok := <-ch; ok {
		t.Fatal("output channel must be closed after the buffer drained")
	}
}

func publishCommandsAfterClose(t *testing.T, bus *command.MemoryBus, cmdType command.Type) {
	t.Helper()

	for range publishesAfterClose {
		cmd, err := command.New(cmdType, id.NewStreamID())
		if err != nil {
			t.Fatalf("command.New(): %v", err)
		}

		_ = bus.Publish(context.Background(), cmd)
	}
}

func TestCommandSubscriberAdapter_PublishAfterCloseDoesNotPanic(t *testing.T) {
	t.Parallel()

	bus := command.NewMemoryBus()
	adapter := NewCommandSubscriberAdapter(bus)

	if _, err := adapter.Subscribe(context.Background(), "test.cmd"); err != nil {
		t.Fatalf("Subscribe(): %v", err)
	}

	if err := adapter.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	publishCommandsAfterClose(t, bus, "test.cmd")
}
