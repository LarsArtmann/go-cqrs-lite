package event_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

var _ = Describe("Event Creation", func() {
	Describe("As a developer creating domain events", func() {
		Context("when I create an event with all metadata", func() {
			It("should preserve every field including tracing IDs", func() {
				aggID := id.NewAggregateID()
				corrID, err := id.ParseCorrelationID("01HK154EJG2GP2SR75DK1Q1TBH")
				Expect(err).ToNot(HaveOccurred())
				causeID, err := id.ParseCausationID("01HK154FHRS5276AC3V7GRNTYM")
				Expect(err).ToNot(HaveOccurred())
				uID, err := id.ParseUserID("01HK1543TRR6BB4AF65NQX5V8S")
				Expect(err).ToNot(HaveOccurred())
				reqID, err := id.ParseRequestID("01HK154HG8WXD9A15YBY6FZJYW")
				Expect(err).ToNot(HaveOccurred())

				evt, err := event.NewEvent(
					event.Type("UserRegistered"),
					aggID,
					id.AggregateType("User"),
					1,
					[]byte(`{"email":"alice@example.com"}`),
					event.WithCorrelationID(corrID),
					event.WithCausationID(causeID),
					event.WithUserID(uID),
					event.WithRequestID(reqID),
					event.WithSource("api"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(evt.Type()).To(Equal(event.Type("UserRegistered")))
				Expect(evt.AggregateID()).To(Equal(aggID))
				Expect(evt.AggregateType()).To(Equal(id.AggregateType("User")))
				Expect(evt.Version()).To(Equal(event.Version(1)))
				Expect(evt.Payload()).To(ContainSubstring("alice@example.com"))
				Expect(evt.Metadata().CorrelationID).To(Equal(corrID))
				Expect(evt.Metadata().CausationID).To(Equal(causeID))
				Expect(evt.Metadata().UserID).To(Equal(uID))
				Expect(evt.Metadata().RequestID).To(Equal(reqID))
				Expect(evt.Metadata().Source).To(Equal(event.Source("api")))
				Expect(evt.OccurredAt()).To(BeTemporally("<", time.Now().Add(time.Second)))
				Expect(evt.ID().IsZero()).To(BeFalse())
			})
		})

		Context("when I create an event with an empty aggregate ID", func() {
			It("should reject it with a descriptive error", func() {
				var emptyID id.AggregateID

				_, err := event.NewEvent(
					event.Type("BadEvent"),
					emptyID,
					id.AggregateType("User"),
					1,
					nil,
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("aggregate ID is required"))
			})
		})

		DescribeTable(
			"when I create an event with invalid parameters",
			func(aggID id.AggregateID, aggType id.AggregateType, version event.Version, expectedMsg string) {
				expectNewEventValidationFails(aggID, aggType, version, expectedMsg)
			},
			Entry(
				"empty aggregate type",
				id.NewAggregateID(),
				id.AggregateType(""),
				event.Version(1),
				"aggregate type is required",
			),
			Entry(
				"zero version",
				id.NewAggregateID(),
				id.AggregateType("User"),
				event.Version(0),
				"version",
			),
		)

		Context("when I add custom metadata to an event", func() {
			It("should preserve it through the metadata map", func() {
				aggID := id.NewAggregateID()
				evt, err := event.NewEvent(
					event.Type("TestEvent"),
					aggID,
					id.AggregateType("Test"),
					1,
					nil,
					event.WithCustom(event.MetadataKey("tenant"), "acme-corp"),
					event.WithCustom(event.MetadataKey("region"), "us-east-1"),
				)
				Expect(err).ToNot(HaveOccurred())

				Expect(
					evt.Metadata().Custom,
				).To(HaveKeyWithValue(event.MetadataKey("tenant"), "acme-corp"))
				Expect(
					evt.Metadata().Custom,
				).To(HaveKeyWithValue(event.MetadataKey("region"), "us-east-1"))
			})
		})
	})
})
