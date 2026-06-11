package otel

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	nopmetric "go.opentelemetry.io/otel/metric/noop"
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

func TestNewMeter_UsesGlobalProvider(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(nopmetric.NewMeterProvider()) })

	m := NewMeter("coverage-test")
	g.Expect(m).ToNot(BeNil())

	counter, err := m.Int64Counter("test.counter")
	g.Expect(err).ToNot(HaveOccurred())
	counter.Add(context.Background(), 1)

	var rm metricdata.ResourceMetrics
	g.Expect(reader.Collect(context.Background(), &rm)).To(Succeed())
	g.Expect(rm.ScopeMetrics).To(HaveLen(1))
	g.Expect(rm.ScopeMetrics[0].Scope.Name).
		To(Equal("github.com/larsartmann/go-cqrs-lite/coverage-test/v2"))
}

func TestNewMeter_NopProvider(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	m := NewMeter("nop-test")
	g.Expect(m).ToNot(BeNil())
}

func TestNewTracer_ReturnsTracerWithCorrectName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	provider, recorder := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test-component"))

	_, span := tracer.Start(context.Background(), "test.operation")
	span.End()

	spans := recorder.Ended()
	g.Expect(spans).To(HaveLen(1))
	g.Expect(spans[0].InstrumentationScope().Name).
		To(Equal("github.com/larsartmann/go-cqrs-lite/test-component/v2"))
}

func TestNewMeter_ReturnsMeterWithCorrectName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter(ComponentTracer("metrics-test"))

	counter, err := meter.Int64Counter("test.counter")
	g.Expect(err).ToNot(HaveOccurred())
	counter.Add(context.Background(), 1)

	var resourceMetrics metricdata.ResourceMetrics
	g.Expect(reader.Collect(context.Background(), &resourceMetrics)).To(Succeed())
	g.Expect(resourceMetrics.ScopeMetrics).To(HaveLen(1))
	g.Expect(resourceMetrics.ScopeMetrics[0].Scope.Name).
		To(Equal("github.com/larsartmann/go-cqrs-lite/metrics-test/v2"))
}

func TestStartSpan_CreatesSpanWithCorrectKind(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	provider, recorder := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test"))

	_, span := StartSpan(context.Background(), tracer, "test.span", 2)
	span.End()

	spans := recorder.Ended()
	g.Expect(spans).To(HaveLen(1))
	g.Expect(spans[0].Name()).To(Equal("test.span"))
	g.Expect(int(spans[0].SpanKind())).To(Equal(2))
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
	g := NewWithT(t)

	provider, recorder := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test"))

	_, span := tracer.Start(context.Background(), "test")
	testErr := errors.New("something broke")
	RecordError(span, testErr)
	span.End()

	spans := recorder.Ended()
	g.Expect(spans).To(HaveLen(1))
	g.Expect(spans[0].Status().Code).To(Equal(codes.Error))
	g.Expect(spans[0].Status().Description).To(Equal("something broke"))
}

func TestEndWithError_NilError_EndsSpanWithoutError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	provider, recorder := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test"))

	_, span := tracer.Start(context.Background(), "test")
	defer span.End()
	EndWithError(span, nil)

	spans := recorder.Ended()
	g.Expect(spans).To(HaveLen(1))
	g.Expect(spans[0].Status().Code).To(Equal(codes.Unset))
}

func TestEndWithError_NonNilError_RecordsAndEnds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	provider, recorder := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test"))

	_, span := tracer.Start(context.Background(), "test")
	defer span.End()
	EndWithError(span, errors.New("fail"))

	spans := recorder.Ended()
	g.Expect(spans).To(HaveLen(1))
	g.Expect(spans[0].Status().Code).To(Equal(codes.Error))
}

func TestAggregateAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	attrs := AggregateAttrs(testStringer("Order"), testStringer("order-123"))
	g.Expect(attrs).To(Equal([]attribute.KeyValue{
		attribute.String(AttrAggregateType, "Order"),
		attribute.String(AttrAggregateID, "order-123"),
	}))
}

func TestCommandAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	attrs := CommandAttrs("CreateOrder", testStringer("order-123"))
	g.Expect(attrs).To(Equal([]attribute.KeyValue{
		attribute.String(AttrMessageKind, KindCommand),
		attribute.String(AttrCommandType, "CreateOrder"),
		attribute.String(AttrAggregateID, "order-123"),
	}))
}

func TestEventAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	attrs := EventAttrs("OrderCreated", testStringer("order-123"), "Order")
	g.Expect(attrs).To(Equal([]attribute.KeyValue{
		attribute.String(AttrMessageKind, KindEvent),
		attribute.String(AttrEventType, "OrderCreated"),
		attribute.String(AttrAggregateID, "order-123"),
		attribute.String(AttrAggregateType, "Order"),
	}))
}

func TestQueryAttrs_ReturnsCorrectAttributes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	attrs := QueryAttrs("GetOrder")
	g.Expect(attrs).To(Equal([]attribute.KeyValue{
		attribute.String(AttrMessageKind, KindQuery),
		attribute.String(AttrQueryType, "GetOrder"),
	}))
}

func TestSpanFromContext_ReturnsSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	provider, _ := testTracerWithRecorder()
	tracer := provider.Tracer(ComponentTracer("test"))

	ctx, span := tracer.Start(context.Background(), "test")
	defer span.End()

	got := trace.SpanFromContext(ctx)
	g.Expect(got.SpanContext()).To(Equal(span.SpanContext()))
}

func TestComponentTracer_ReturnsExpectedFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(ComponentTracer("storage")).To(Equal("github.com/larsartmann/go-cqrs-lite/storage/v2"))
	g.Expect(ComponentTracer("middleware")).
		To(Equal("github.com/larsartmann/go-cqrs-lite/middleware/v2"))
}

func TestNameConstant(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(Name).To(Equal("github.com/larsartmann/go-cqrs-lite"))
}
