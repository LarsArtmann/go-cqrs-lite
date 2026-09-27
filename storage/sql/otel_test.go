package sql

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type customDialect struct{ SQLiteDialect }

// fixedString satisfies fmt.Stringer for id.StreamIDFrom.
type fixedString string

func (f fixedString) String() string { return string(f) }

func recordedSpan(
	t *testing.T,
	start func(context.Context) (context.Context, cqrsotel.Span),
) tracetest.SpanStub {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	prevTP := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter)))
	t.Cleanup(func() { otel.SetTracerProvider(prevTP) })

	_, span := start(context.Background())
	span.End()

	stubs := exporter.GetSpans()
	if len(stubs) != 1 {
		t.Fatalf("recorded %d spans, want 1", len(stubs))
	}

	return stubs[0]
}

func dbSystemAttr(stub tracetest.SpanStub) (attribute.Value, bool) {
	for _, kv := range stub.Attributes {
		if kv.Key == "db.system" {
			return kv.Value, true
		}
	}

	return attribute.Value{}, false
}

func TestStartDialectSpanStampsDBSystem(t *testing.T) {
	cases := []struct {
		name    string
		dialect Dialect
		want    string
	}{
		{"sqlite", SQLiteDialect{}, "sqlite"},
		{"postgres", PostgresDialect{}, "postgresql"},
		{"mysql", MySQLDialect{}, "mysql"},
		{"duckdb", DuckDBDialect{}, "duckdb"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := recordedSpan(t, func(ctx context.Context) (context.Context, cqrsotel.Span) {
				return StartDialectSpan(ctx, "test.op", tc.dialect)
			})

			got, ok := dbSystemAttr(stub)
			if !ok || got.AsString() != tc.want {
				t.Fatalf("db.system = %v (present=%v), want %q", got, ok, tc.want)
			}
		})
	}
}

func TestStartDialectSpanUnknownDialectUnstamped(t *testing.T) {
	stub := recordedSpan(t, func(ctx context.Context) (context.Context, cqrsotel.Span) {
		return StartDialectSpan(ctx, "test.op", customDialect{})
	})

	if _, ok := dbSystemAttr(stub); ok {
		t.Fatal("custom dialect must not carry a guessed db.system attribute")
	}
}

func TestStartStreamAndSaveSpanWithDialect(t *testing.T) {
	ref := id.NewStreamRef("user", id.StreamIDFrom(fixedString("u1")))

	stub := recordedSpan(t, func(ctx context.Context) (context.Context, cqrsotel.Span) {
		return StartStreamSpanWithDialect(ctx, "test.stream", PostgresDialect{}, ref)
	})

	if got, ok := dbSystemAttr(stub); !ok || got.AsString() != "postgresql" {
		t.Fatalf("stream span db.system = %v (present=%v), want postgresql", got, ok)
	}

	stub = recordedSpan(t, func(ctx context.Context) (context.Context, cqrsotel.Span) {
		return StartSaveSpanWithDialect(ctx, "test.save", SQLiteDialect{}, ref, 3, 2)
	})

	if got, ok := dbSystemAttr(stub); !ok || got.AsString() != "sqlite" {
		t.Fatalf("save span db.system = %v (present=%v), want sqlite", got, ok)
	}
}
