package event_test

import (
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func createTestEvent(
	eventType event.Type,
	streamID id.StreamID,
	version event.Version,
	payload []byte,
) event.Event {
	if payload == nil {
		payload = []byte(`{"test":true}`)
	}

	evt, err := event.NewEvent(eventType, streamID, "TestStream", version, payload)
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func expectNewEventValidationFails(
	streamID id.StreamID,
	streamType id.StreamType,
	version event.Version,
	expectedMsg string,
) {
	_, err := event.NewEvent(event.Type("BadEvent"), streamID, streamType, version, nil)
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(expectedMsg))
}

func subscribeOrderPlaced(bus event.Bus, received *[]event.Event) error {
	return bus.Subscribe(
		event.Type("OrderPlaced"),
		eventtest.AppendEventsHandler(received),
	)
}
