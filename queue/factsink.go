package queue

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
)

// FactSink appends caller-authored facts to the journal. A sink handed
// out by FactTx.WithFacts is transactional: its appends commit or roll
// back with the surrounding transaction, never on their own.
type FactSink interface {
	// Append records one fact. Seq and Time are assigned by the journal;
	// TaskID and Type are the caller's attribution and ride verbatim.
	Append(ctx context.Context, f facts.Fact) error
}

// FactTx is the same-transaction fact-append capability (the ADR-0001
// lineage, upstreamed): WithFacts runs fn inside ONE transaction whose
// FactSink appends land in the journal exactly when fn returns nil — an
// error from fn rolls every append back. This is the seam for consumer
// observations (progress markers, external-effect evidence, audit notes)
// that must never disagree with the journal, with the same atomicity the
// store's own state changes enjoy.
//
// Engines provide it over the SAME connection domain as the task tables
// (queue/sqlite: the single-writer connection; queue/postgres: the
// pool); the shared conformance suite pins the commit/rollback split.
type FactTx interface {
	WithFacts(ctx context.Context, fn func(sink FactSink) error) error
}
