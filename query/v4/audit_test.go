package query_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

type fakeQuerySink struct {
	saved []*query.PersistedQuery
}

func (f *fakeQuerySink) SaveQuery(_ context.Context, q *query.PersistedQuery) error {
	f.saved = append(f.saved, q)

	return nil
}

func (f *fakeQuerySink) Close() error { return nil }

func TestAuditMiddleware_Off(t *testing.T) {
	t.Parallel()

	sink := &fakeQuerySink{}
	d := query.NewDispatcher()
	d.Use(query.AuditMiddleware(sink, query.AuditOff, nil))

	_ = d.Register("test.q", func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	})

	q, _ := query.New("test.q")

	_, _ = d.Dispatch(context.Background(), q)

	if len(sink.saved) != 0 {
		t.Fatalf("AuditOff should not persist, got %d records", len(sink.saved))
	}
}

func TestAuditMiddleware_Metadata(t *testing.T) {
	t.Parallel()

	sink := &fakeQuerySink{}
	d := query.NewDispatcher()
	d.Use(query.AuditMiddleware(sink, query.AuditMetadata, nil))

	_ = d.Register("test.q", func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	})

	q, _ := query.New("test.q")

	_, _ = d.Dispatch(context.Background(), q)

	if len(sink.saved) != 1 {
		t.Fatalf("expected 1 persisted query, got %d", len(sink.saved))
	}

	pq := sink.saved[0]
	if pq.Type() != "test.q" {
		t.Fatalf("expected type test.q, got %s", pq.Type())
	}

	if pq.ReceivedAt().IsZero() {
		t.Fatal("expected non-zero ReceivedAt")
	}

	if len(pq.Payload()) != 0 {
		t.Fatal("AuditMetadata should not save payload")
	}
}

func TestAuditMiddleware_NilSink(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	d.Use(query.AuditMiddleware(nil, query.AuditFull, nil))

	_ = d.Register("test.q", func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	})

	q, _ := query.New("test.q")

	_, err := d.Dispatch(context.Background(), q)
	if err != nil {
		t.Fatalf("Dispatch with nil sink should succeed: %v", err)
	}
}

var (
	_ io.Closer = (*fakeQuerySink)(nil)
	_ time.Duration
)

// TestAuditMiddleware_CarriesRequestIDAndMetadata pins the correlation fix:
// audit records must carry the query's REAL request ID and its metadata
// (correlation, causation, actor, custom), not a freshly minted stripped ID.
func TestAuditMiddleware_CarriesRequestIDAndMetadata(t *testing.T) {
	t.Parallel()

	sink := &fakeQuerySink{}
	d := query.NewDispatcher()
	d.Use(query.AuditMiddleware(sink, query.AuditFull, nil))

	_ = d.Register("test.q", func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	})

	requestID := id.NewRequestID()
	correlationID := id.NewCorrelationID()

	q, err := query.New("test.q",
		query.WithRequestID(requestID),
		query.WithCorrelationID(correlationID),
		query.WithCustomMetadata("tenant", "acme"),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, _ = d.Dispatch(context.Background(), q)

	if len(sink.saved) != 1 {
		t.Fatalf("expected 1 persisted query, got %d", len(sink.saved))
	}

	pq := sink.saved[0]

	if pq.ID() != requestID {
		t.Errorf("audit ID = %v, want the query's real request ID %v", pq.ID(), requestID)
	}

	md := pq.Metadata()
	if md.RequestID != requestID {
		t.Errorf("audit metadata RequestID = %v, want %v", md.RequestID, requestID)
	}

	if md.CorrelationID != correlationID {
		t.Errorf("audit metadata CorrelationID = %v, want %v", md.CorrelationID, correlationID)
	}

	if md.Custom["tenant"] != "acme" {
		t.Errorf(`audit metadata Custom["tenant"] = %q, want "acme"`, md.Custom["tenant"])
	}
}
