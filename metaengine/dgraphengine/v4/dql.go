package dgraphengine

// DQL tokens shared by every builder: the $col query variable, the _:new
// blank node for upsert-style mutations, and the dgraph.type predicate that
// stamps node types. Query templates reference the tokens textually; these
// constants keep the Go-side var maps and mutation scaffolding in sync.
const (
	dqlColVar   = "$col"
	dqlNewNode  = "_:new"
	dqlTypePred = "dgraph.type"
	dqlKeyVar   = "$key"
)

// colVars builds the variable map every $col-scoped query passes to
// QueryWithVars — one construction site for the shared variable name.
func colVars(col string) map[string]string {
	return map[string]string{dqlColVar: col}
}
