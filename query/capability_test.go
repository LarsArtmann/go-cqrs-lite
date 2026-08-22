package query_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// handRolledMetadataQuery proves a consumer implementation OUTSIDE this repo
// satisfies query.MetadataCarrier without embedding BasicQuery — the whole
// point of the capability pattern (plan Appendix C).
type handRolledMetadataQuery struct {
	requestID id.RequestID
}

func (h *handRolledMetadataQuery) Type() query.Type { return "user.by-id" }

func (h *handRolledMetadataQuery) Metadata() query.Metadata {
	return query.Metadata{
		Tracing: metadata.Tracing{RequestID: h.requestID},
	}
}

var (
	_ query.Query           = (*handRolledMetadataQuery)(nil)
	_ query.MetadataCarrier = (*handRolledMetadataQuery)(nil)
)

func TestMetadataCarrier_SatisfiedByHandRolledImplementation(t *testing.T) {
	t.Parallel()

	var q query.Query = &handRolledMetadataQuery{requestID: id.NewRequestID()}

	carrier, ok := q.(query.MetadataCarrier)
	if !ok {
		t.Fatal("hand-rolled Query does not satisfy MetadataCarrier")
	}

	if carrier.Metadata().RequestID.IsZero() {
		t.Fatal("MetadataCarrier returned empty request ID")
	}
}

func TestMetadataCarrier_BasicQuerySatisfies(t *testing.T) {
	t.Parallel()

	basic, err := query.New("user.by-id")
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	if _, ok := any(basic).(query.MetadataCarrier); !ok {
		t.Fatal("*BasicQuery must satisfy MetadataCarrier")
	}
}

func TestPayloadCarrier_NotSatisfiedByPayloadlessQuery(t *testing.T) {
	t.Parallel()

	basic, err := query.New("user.by-id")
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	if _, ok := any(basic).(query.PayloadCarrier); ok {
		t.Fatal("payload-less BasicQuery must not satisfy PayloadCarrier")
	}
}
