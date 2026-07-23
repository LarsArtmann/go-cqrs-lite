package event_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

var _ = Describe("Event Metadata Roundtrip", func() {
	var (
		ctx   context.Context
		store *memory.MemoryStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = memory.NewMemoryStore()
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	It("preserves all metadata fields through MemoryStore Save+Load", func() {
		aggID := id.NewStreamID()
		corrID := id.NewCorrelationID()
		causID := id.NewCausationID()
		userID := id.NewUserID()
		reqID := id.NewRequestID()

		evt, err := event.NewEvent(
			"user.created",
			aggID,
			"User",
			1,
			[]byte(`{"name":"Alice"}`),
			event.WithCorrelationID(corrID),
			event.WithCausationID(causID),
			event.WithUserID(userID),
			event.WithRequestID(reqID),
			event.WithSource("test-service"),
			event.WithIPAddress("192.168.1.1"),
			event.WithUserAgent("TestAgent/1.0"),
			event.WithCustom("trace-id", "abc-123"),
			event.WithCustom("span-id", "def-456"),
		)
		Expect(err).ToNot(HaveOccurred())

		err = store.Save(
			ctx,
			id.NewStreamRef(id.StreamType("User"), aggID),
			[]event.Event{evt},
			event.Version(0),
		)
		Expect(err).ToNot(HaveOccurred())

		loaded, err := store.Load(ctx, id.NewStreamRef(id.StreamType("User"), aggID))
		Expect(err).ToNot(HaveOccurred())
		Expect(loaded).To(HaveLen(1))

		got := loaded[0]
		Expect(got.ID()).To(Equal(evt.ID()))
		Expect(got.Type()).To(Equal(event.Type("user.created")))
		Expect(got.StreamID()).To(Equal(aggID))
		Expect(got.StreamType()).To(Equal(id.StreamType("User")))
		Expect(got.Version()).To(Equal(event.Version(1)))

		meta := got.Metadata()
		Expect(meta).ToNot(BeNil())
		Expect(meta.CorrelationID).To(Equal(corrID))
		Expect(meta.CausationID).To(Equal(causID))
		Expect(meta.UserID).To(Equal(userID))
		Expect(meta.RequestID).To(Equal(reqID))
		Expect(meta.Source).To(Equal(event.Source("test-service")))
		Expect(meta.IPAddress).To(Equal(event.IPAddress("192.168.1.1")))
		Expect(meta.UserAgent).To(Equal(event.UserAgent("TestAgent/1.0")))
		Expect(meta.Custom).To(HaveLen(2))
		Expect(meta.Custom["trace-id"]).To(Equal("abc-123"))
		Expect(meta.Custom["span-id"]).To(Equal("def-456"))
	})

	It("preserves payload bytes through MemoryStore Save+Load", func() {
		aggID := id.NewStreamID()
		payload := []byte(`{"complex":"data","nested":{"key":"value"}}`)

		evt, err := event.NewEvent("test.event", aggID, "Test", 1, payload)
		Expect(err).ToNot(HaveOccurred())

		err = store.Save(
			ctx,
			id.NewStreamRef(id.StreamType("Test"), aggID),
			[]event.Event{evt},
			event.Version(0),
		)
		Expect(err).ToNot(HaveOccurred())

		loaded, err := store.Load(ctx, id.NewStreamRef(id.StreamType("Test"), aggID))
		Expect(err).ToNot(HaveOccurred())
		Expect(loaded[0].Payload()).To(Equal(payload))
		Expect(loaded[0].OccurredAt()).To(BeTemporally("~", evt.OccurredAt(), 0))
	})

	It("preserves metadata through LoadFromVersion", func() {
		aggID := id.NewStreamID()
		corrID := id.NewCorrelationID()
		aggType := id.StreamType("Test")

		var events []event.Event

		for i := 1; i <= 3; i++ {
			evt, err := event.NewEvent(
				"test.event",
				aggID,
				"Test",
				event.Version(i),
				[]byte(`{}`),
				event.WithCorrelationID(corrID),
			)
			Expect(err).ToNot(HaveOccurred())

			events = append(events, evt)
		}

		Expect(
			store.Save(ctx, id.NewStreamRef(aggType, aggID), events, event.Version(0)),
		).To(Succeed())

		loaded, err := store.LoadFromVersion(
			ctx,
			id.NewStreamRef(aggType, aggID),
			event.Version(1),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(loaded).To(HaveLen(2))
		Expect(loaded[0].Version()).To(Equal(event.Version(2)))
		Expect(loaded[1].Version()).To(Equal(event.Version(3)))

		for _, evt := range loaded {
			Expect(evt.Metadata().CorrelationID).To(Equal(corrID))
		}
	})
})
