package grpc

import (
	"context"
	"fmt"

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
		return fmt.Errorf("grpc: open event stream: %w", err)
	}

	for {
		envelope, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("grpc: receive event: %w", err)
		}

		evt, err := envelopeToEvent(envelope)
		if err != nil {
			return fmt.Errorf("grpc: decode event: %w", err)
		}

		err = handler(ctx, evt)
		if err != nil {
			return fmt.Errorf("grpc: event handler: %w", err)
		}
	}
}

func envelopeToEvent(envelope *cqrsproto.EventEnvelope) (event.Event, error) {
	aggID, err := id.ParseAggregateID(envelope.GetAggregateId())
	if err != nil {
		return nil, fmt.Errorf("parse aggregate ID: %w", err)
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
		return nil, fmt.Errorf("reconstruct event: %w", err)
	}

	return evt, nil
}
