// Package search is the search contract of go-cqrs-lite: one call that serves
// every kind of search and reports which kind ran.
//
// An author declares the searchable fields of a record type once, as a
// [Spec], and builds an [Index] with [New]. The index reads records through a
// [Source] and asks an [Engine] which kinds of search the deployment offers.
// Operators choose the engine and, if they want semantic search, an
// [Embedder]. Neither side needs to know which kind ran, and [Result.Via]
// names it anyway.
//
// Search is a ladder of kinds. Mode Auto climbs it as far as the deployment
// and the query allow:
//
//	Tier  Kind        Needs                        Reported as
//	0     Client      a static index               never server-side
//	1     Scan        nothing                      ViaScan
//	2     Full-text   a full-text index            ViaFullText
//	3     Vector      a vector index and embedder  ViaVector
//	4     Hybrid      tiers 2 and 3                ViaHybrid
//
// Facets ([Request.Filters]) combine with every tier. An explicit [Mode] is
// strict: asking for a kind the deployment cannot serve is an error, never a
// silent step down.
//
// The SDK never embeds on its own. A nil embedder means lexical search only,
// which is a complete product.
//
// No function in this package returns HTML. Highlighting is data:
// [Snippet] returns [Segment] values that mark which runs matched, and the
// template picks the tag.
//
// This first release carries the lexical rungs: the in-memory ranking rule
// ([Score]), the query helpers ([Tokenize], [LikePattern], [FTS5Query]), the
// snippet window ([Snippet]) and the static index ([BuildStaticIndex]).
// [Score] rates about 150,000 records per second for three terms over two
// fields on one core of a Ryzen AI Max+ 395 (2026-11-28, BenchmarkScore);
// past roughly a hundred thousand records, give the engine a full-text index.
package search
