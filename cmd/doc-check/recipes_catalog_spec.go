package main

// recipeSpec decides how the compile gate treats one recipes.md block.
//
// Every block in recipes.md MUST have exactly one catalog entry (in
// recipeCatalogA, recipeCatalogB, or recipeCatalogB2): either a compiled
// scaffold (imports, preamble for free variables, trailers blanking unused
// locals) or a skip with a reason. A new or reworded section therefore fails
// the coverage test until it is classified — snippets can no longer rot
// silently.
type recipeSpec struct {
	skip         string   // non-empty: documented reason the block is not compiled
	imports      []string // raw import lines (aliases/blank imports allowed)
	preamble     string   // declarations placed above func main (free vars, types)
	trailers     string   // statements appended inside main to blank unused locals
	wholeProgram bool     // block already starts with "package " — use verbatim
	errFunc      bool     // wrap statements in `func run() error` (block returns errors)
}
