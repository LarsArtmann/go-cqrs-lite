package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// handRolledMetadataQuery proves a consumer implementation OUTSIDE this repo
// satisfies query.MetadataCarrier without embedding BasicQuery — the whole
// point of the capability pattern (plan Appendix C).
type handRolledMetadataQuery struct {
	queryType query.Type
	requestID id.RequestID
}

func (h *handRolledMetadataQuery) Type() query.Type { return h.queryType }

func (h *handRolledMetadataQuery) Metadata() query.Metadata {
	return query.Metadata{Tracing: struct {
		CorrelationID id.CorrelationID "json:\"correlationId,omitempty\""
		CausationID   id.CausationID   "json:\"causationId,omitempty\""
		RequestID     id.RequestID     "json:\"requestId,omitempty\""
		UserID        id.UserID        "json:\"userId,omitempty\""
		ActorID       id.ActorID       "json:\"actorId,omitempty\""
	}{RequestID: h.requestID}}
}

var (
	_ query.Query           = (*handRolledMetadataQuery)(nil)
	_ query.MetadataCarrier = (*handRolledMetadataQuery)(nil)
)

func TestMetadataCarrier_SatisfiedByHandRolledImplementation(t *testing.T) {
	t.Parallel()

	var q query.Query = &handRolledMetadataQuery{
		queryType: "user.by-id",
	}

	carrier, ok := q.(query.MetadataCarrier)
	if !ok {
		t.Fatal("hand-rolled Query does not satisfy MetadataCarrier")
	}

	if carrier.Metadata().Tracing.RequestID.String() == "" {
		t.Fatal("MetadataCarrier returned empty request ID")
	}
}

func TestPayloadCarrier_InterfaceSatisfiable(t *testing.T) {
	t.Parallel()

	// Compile-time proof the capability interface is the contract the audit
	// middleware asserts; payload-carrying queries implement it, payload-less
	// queries simply do not.
	type payloadQuery struct {
		*query.BasicQuery
	}

	var _ query.Query = payloadQuery{}
	if _, ok := any(payloadQuery{}).(query.PayloadCarrier); ok {
		t.Fatal("BasicQuery embed without Payload() must not satisfy PayloadCarrier")
	}
}

func TestAuditMiddleware_UsesCapabilityForRequestID(t *testing.T) {
	t.Parallel()

	rid := id.NewRequestID()
	q := &handRolledMetadataQuery{queryType: "user.by-id", requestID: rid}

	if got := requestIDOfForTest(q); got != rid {
		t.Fatalf("requestIDOf via capability = %v, want %v", got, rid)
	}
}

func requestIDOfForTest(q query.Query) id.RequestID { return requestIDOf(q) }

var _ = context.Background
