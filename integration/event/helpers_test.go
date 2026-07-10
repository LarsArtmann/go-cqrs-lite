package event_test

import (
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func createTestEvent(
	eventType event.Type,
	aggID id.AggregateID,
	version event.Version,
	payload []byte,
) event.Event {
	if payload == nil {
		payload = []byte(`{"test":true}`)
	}

	evt, err := event.NewEvent(eventType, aggID, "TestAggregate", version, payload)
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func expectNewEventValidationFails(
	aggID id.AggregateID,
	aggType id.AggregateType,
	version event.Version,
	expectedMsg string,
) {
	_, err := event.NewEvent(event.Type("BadEvent"), aggID, aggType, version, nil)
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(expectedMsg))
}

func subscribeOrderPlaced(bus event.Bus, received *[]event.Event) error {
	return bus.Subscribe(
		event.Type("OrderPlaced"),
		eventtest.AppendEventsHandler(received),
	)
}
