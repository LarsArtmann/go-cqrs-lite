package otel

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type testStringer string

func (s testStringer) String() string { return string(s) }

func testTracerWithRecorder() (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	return provider, recorder
}

func withGlobalProvider(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(otel.GetTracerProvider())
	})

	return recorder
}

func TestNewTracer_ReturnsTracerWithCorrectName(t *testing.T) {
	t.Parallel()

	provider, recorder := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test-component"))

	_, span := tracer.Start(context.Background(), "test.operation")
	span.End()

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(
		t,
		"github.com/larsartmann/go-cqrs-lite/test-component",
		spans[0].InstrumentationScope().Name,
	)
}

func TestNewMeter_ReturnsMeterWithCorrectName(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter(ComponentTracer("metrics-test"))

	counter, err := meter.Int64Counter("test.counter")
	require.NoError(t, err)
	counter.Add(context.Background(), 1)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	require.Len(t, rm.ScopeMetrics, 1)
	require.Equal(
		t,
		"github.com/larsartmann/go-cqrs-lite/metrics-test",
		rm.ScopeMetrics[0].Scope.Name,
	)
}

func TestStartSpan_CreatesSpanWithCorrectKind(t *testing.T) {
	t.Parallel()

	provider, recorder := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test"))

	_, span := StartSpan(context.Background(), tracer, "test.span", 2)
	span.End()

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "test.span", spans[0].Name())
	require.Equal(t, 2, int(spans[0].SpanKind()))
}

func TestStartSpan_NilProvider_NoPanic(t *testing.T) {
	t.Parallel()

	tr := NewTracer("test")
	ctx, span := StartSpan(context.Background(), tr, "noop", 1)
	defer span.End()

	sc := trace.SpanFromContext(ctx).SpanContext()
	_ = sc
}

func TestRecordError_SetsErrorStatus(t *testing.T) {
	t.Parallel()

	recorder := withGlobalProvider(t)
	tracer := NewTracer("test")

	_, span := tracer.Start(context.Background(), "test")
	testErr := errors.New("something broke") //nolint:err113 // test error
	RecordError(span, testErr)
	span.End()

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Error, spans[0].Status().Code)
	require.Equal(t, "something broke", spans[0].Status().Description)
}

func TestEndWithError_NilError_EndsSpanWithoutError(t *testing.T) {
	t.Parallel()

	recorder := withGlobalProvider(t)
	tracer := NewTracer("test")

	_, span := tracer.Start(context.Background(), "test")
	EndWithError(span, nil)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Unset, spans[0].Status().Code)
}

func TestEndWithError_NonNilError_RecordsAndEnds(t *testing.T) {
	t.Parallel()

	recorder := withGlobalProvider(t)
	tracer := NewTracer("test")

	_, span := tracer.Start(context.Background(), "test")
	EndWithError(span, errors.New("fail")) //nolint:err113 // test error

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Error, spans[0].Status().Code)
}

func TestAggregateAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()

	attrs := AggregateAttrs(testStringer("Order"), testStringer("order-123"))
	require.Equal(t, []attribute.KeyValue{
		attribute.String(AttrAggregateType, "Order"),
		attribute.String(AttrAggregateID, "order-123"),
	}, attrs)
}

func TestCommandAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()

	attrs := CommandAttrs("CreateOrder", testStringer("order-123"))
	require.Equal(t, []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindCommand),
		attribute.String(AttrCommandType, "CreateOrder"),
		attribute.String(AttrAggregateID, "order-123"),
	}, attrs)
}

func TestEventAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()

	attrs := EventAttrs("OrderCreated", testStringer("order-123"), "Order")
	require.Equal(t, []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindEvent),
		attribute.String(AttrEventType, "OrderCreated"),
		attribute.String(AttrAggregateID, "order-123"),
		attribute.String(AttrAggregateType, "Order"),
	}, attrs)
}

func TestQueryAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()

	attrs := QueryAttrs("GetOrder")
	require.Equal(t, []attribute.KeyValue{
		attribute.String(AttrMessageKind, KindQuery),
		attribute.String(AttrQueryType, "GetOrder"),
	}, attrs)
}

func TestSpanFromContext_ReturnsSpan(t *testing.T) {
	t.Parallel()

	provider, _ := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test"))

	ctx, span := tracer.Start(context.Background(), "test")
	defer span.End()

	got := SpanFromContext(ctx)
	require.Equal(t, span.SpanContext(), got.SpanContext())
}

func TestComponentTracer_ReturnsExpectedFormat(t *testing.T) {
	t.Parallel()

	require.Equal(t, "github.com/larsartmann/go-cqrs-lite/storage", ComponentTracer("storage"))
	require.Equal(
		t,
		"github.com/larsartmann/go-cqrs-lite/middleware",
		ComponentTracer("middleware"),
	)
}

func TestNameConstant(t *testing.T) {
	t.Parallel()

	require.Equal(t, "github.com/larsartmann/go-cqrs-lite", Name)
}
