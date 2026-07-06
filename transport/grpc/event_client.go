package grpc

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"
	"google.golang.org/grpc"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsproto "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3/proto"
)

// EventClient subscribes to events from a remote gRPC EventService.
// It opens a server-streaming connection and delivers events to a handler.
type EventClient struct {
	client cqrsproto.EventServiceClient
}

// NewEventClient creates an EventClient from an existing gRPC connection.
func NewEventClient(conn *grpc.ClientConn) *EventClient {
	return &EventClient{client: cqrsproto.NewEventServiceClient(conn)}
}

// Subscribe opens a streaming subscription to the remote event service.
// It blocks until ctx is cancelled or the stream breaks. Each received event
// is delivered to handler. If eventTypes is non-empty, only matching events
// are streamed from the server.
func (c *EventClient) Subscribe(
	ctx context.Context,
	handler event.Handler,
	eventTypes ...string,
) error {
	stream, err := c.client.Subscribe(ctx, &cqrsproto.SubscriptionRequest{
		EventTypes: eventTypes,
	})
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "grpc.event_client.open_stream",
			"open event stream")
	}

	for {
		envelope, err := stream.Recv()
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "grpc.event_client.receive",
				"receive event")
		}

		evt, err := envelopeToEvent(envelope)
		if err != nil {
			return errorfamily.WrapCorruption(err, "grpc.event_client.decode",
				"decode event")
		}

		err = handler(ctx, evt)
		if err != nil {
			return errorfamily.Wrap(err, errorfamily.Classify(err),
				"grpc.event_client.handler", "event handler")
		}
	}
}

func envelopeToEvent(envelope *cqrsproto.EventEnvelope) (event.Event, error) {
	aggID, err := id.ParseAggregateID(envelope.GetAggregateId())
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "grpc.event_client.parse_aggregate_id",
			"parse aggregate ID")
	}

	var opts []event.Option

	corrID := envelope.GetMetadata()["correlation_id"]

	if corrID != "" {
		parsed, parseErr := id.ParseCorrelationID(corrID)
		if parseErr == nil {
			opts = append(opts, event.WithCorrelationID(parsed))
		}
	}

	evt, err := event.NewEvent(
		event.Type(envelope.GetType()),
		aggID,
		event.AggregateType(envelope.GetAggregateType()),
		safeVersionFromInt64(envelope.GetVersion()),
		envelope.GetPayload(),
		opts...,
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "grpc.event_client.reconstruct",
			"reconstruct event")
	}

	return evt, nil
}
