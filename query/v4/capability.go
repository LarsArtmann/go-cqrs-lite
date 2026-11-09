package query

// MetadataCarrier is the capability interface for queries that carry
// [Metadata]. Embedding [BasicQuery] satisfies it; hand-rolled [Query]
// implementations opt in by adding a Metadata() method.
//
// Middleware that needs request-scoped context (audit trails, correlation,
// actor attribution) type-asserts Query to MetadataCarrier instead of
// growing the exported Query interface — adding methods to Query would
// break every existing hand-rolled implementation, so interface growth
// rides the v5 cut (decision recorded 2026-08-22, plan Appendix C).
type MetadataCarrier interface {
	Metadata() Metadata
}

// PayloadCarrier is the capability interface for queries that expose their
// raw payload bytes (e.g. transport-decoded queries). AuditMiddleware uses
// it at AuditFull level to persist the query payload without requiring
// payload support on the core Query interface.
type PayloadCarrier interface {
	Payload() []byte
}

var _ MetadataCarrier = (*BasicQuery)(nil)
