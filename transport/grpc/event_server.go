package grpc

import (
	"context"
	"fmt"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
	"google.golang.org/grpc"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsproto "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4/proto"
)

// EventSubscriber is the server-side interface that the gRPC event service
// subscribes to. [event.Bus] satisfies this interface.
type EventSubscriber interface {
	SubscribeAll(handler event.Handler) error
}

// RegisterEventService registers a CQRS event streaming service on the given
// gRPC server. Remote clients can subscribe to the event stream via a
// server-streaming RPC.
//
// The server subscribes to ALL events on the bus and fans them out to each
// connected client. Clients can filter by event type via SubscriptionRequest.
func RegisterEventService(srv *grpc.Server, bus EventSubscriber) (*EventServer, error) {
	eventSrv := &EventServer{ //nolint:exhaustruct // grpc server pattern
		clients: make(map[int64]chan *cqrsproto.EventEnvelope),
		nextID:  1,
	}

	err := bus.SubscribeAll(eventSrv.handleEvent)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "grpc.event_server.subscribe",
			"subscribe to event bus")
	}

	cqrsproto.RegisterEventServiceServer(srv, eventSrv)

	return eventSrv, nil
}

// EventServer adapts an event.Bus to a gRPC server-streaming service.
type EventServer struct {
	cqrsproto.UnimplementedEventServiceServer

	mu      sync.RWMutex
	clients map[int64]chan *cqrsproto.EventEnvelope
	nextID  int64
}

const clientBufferSize = 128

// Subscribe handles a client subscription. Events are streamed until the
// client disconnects (context cancelled) or the server shuts down.
func (s *EventServer) Subscribe(
	req *cqrsproto.SubscriptionRequest,
	stream grpc.ServerStreamingServer[cqrsproto.EventEnvelope],
) error {
	ch := s.registerClient()
	defer s.unregisterClient(ch)

	filter := make(map[string]struct{}, len(req.GetEventTypes()))

	for _, t := range req.GetEventTypes() {
		filter[t] = struct{}{}
	}

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return nil
		case envelope := <-ch:
			if len(filter) > 0 {
				_, ok := filter[envelope.GetType()]
				if !ok {
					continue
				}
			}

			err := stream.Send(envelope)
			if err != nil {
				return errorfamily.WrapInfrastructure(err, "grpc.event_server.send",
					"send event")
			}
		}
	}
}

func (s *EventServer) handleEvent(_ context.Context, evt event.Event) error {
	envelope := eventToEnvelope(evt)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.clients {
		select {
		case ch <- envelope:
		default: // drop if client buffer full (slow consumer)
		}
	}

	return nil
}

func (s *EventServer) registerClient() chan *cqrsproto.EventEnvelope {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.nextID++

	ch := make(chan *cqrsproto.EventEnvelope, clientBufferSize)
	s.clients[id] = ch

	return ch
}

func (s *EventServer) unregisterClient(ch chan *cqrsproto.EventEnvelope) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, c := range s.clients {
		if c == ch {
			close(ch)
			delete(s.clients, id)

			return
		}
	}
}

const estimatedMetadataEntries = 4

func eventToEnvelope(evt event.Event) *cqrsproto.EventEnvelope {
	md := evt.Metadata()

	meta := make(map[string]string, estimatedMetadataEntries)

	corrID := md.CorrelationID.String()
	if corrID != "" {
		meta["correlation_id"] = corrID
	}

	if md.Causation != nil {
		meta["command_type"] = md.Causation.CommandType
		meta["command_id"] = md.Causation.CommandID.String()
	}

	if md.Tombstone != nil {
		meta["tombstone_status"] = fmt.Sprintf("%d", md.Tombstone.Status)
	}

	for k, v := range md.Custom {
		meta[string(k)] = v
	}

	version := safeInt64FromVersion(evt.Version())

	return &cqrsproto.EventEnvelope{
		Id:                 evt.ID().String(),
		Type:               string(evt.Type()),
		AggregateId:        evt.AggregateID().String(),
		AggregateType:      string(evt.AggregateType()),
		Version:            version,
		Payload:            event.PayloadReadOnly(evt),
		OccurredAtUnixNano: evt.OccurredAt().UnixNano(),
		Metadata:           meta,
	}
}
