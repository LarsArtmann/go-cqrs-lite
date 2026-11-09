package command

// MetadataCarrier is the capability interface for commands that carry
// [Metadata]. Embedding [BasicCommand] satisfies it; hand-rolled [Command]
// implementations opt in by adding a Metadata() method.
//
// Middleware that needs request-scoped context (audit trails, correlation,
// actor attribution) type-asserts Command to MetadataCarrier instead of
// growing the exported Command interface — adding methods to Command would
// break every existing hand-rolled implementation, so interface growth
// rides the v5 cut (decision recorded 2026-08-22, plan Appendix C).
type MetadataCarrier interface {
	Metadata() Metadata
}

var _ MetadataCarrier = (*BasicCommand)(nil)
