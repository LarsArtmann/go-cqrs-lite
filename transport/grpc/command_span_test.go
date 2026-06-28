package grpc_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3"
)

func TestCommandDispatch_ServerSpanCarriesAttrs(t *testing.T) {
	// NOT parallel — mutates global TracerProvider.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(origTP)

	lis := listen(t)

	srv := grpc.NewServer()

	cmdDispatcher := command.NewDispatcher()
	_ = cmdDispatcher.Register("test.cmd", func(_ context.Context, _ command.Command) error {
		return nil
	})

	cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer conn.Close()

	client := cqrsgrpc.NewCommandClient(conn)

	aggID := id.NewAggregateID()
	cmd, err := command.New("test.cmd", aggID)
	if err != nil {
		t.Fatalf("New command: %v", err)
	}

	if err := client.Dispatch(context.Background(), cmd); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans()

	var dispatch *tracetest.SpanStub

	for i := range spans {
		if spans[i].Name == "grpc.command.dispatch" {
			dispatch = &spans[i]
		}
	}

	if dispatch == nil {
		names := make([]string, 0, len(spans))
		for _, s := range spans {
			names = append(names, s.Name)
		}
		t.Fatalf("grpc.command.dispatch span not found, got: %v", names)
	}

	attrs := make(map[string]string, len(dispatch.Attributes))
	for _, kv := range dispatch.Attributes {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}

	if attrs[cqrsotel.AttrCommandType] != "test.cmd" {
		t.Errorf("expected command type attr = test.cmd, got %v", attrs[cqrsotel.AttrCommandType])
	}

	if attrs[cqrsotel.AttrAggregateID] != aggID.String() {
		t.Errorf("expected aggregate ID attr = %s, got %v", aggID.String(), attrs[cqrsotel.AttrAggregateID])
	}
}
