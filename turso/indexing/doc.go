package indexing

// Package indexing provides auto-smart index management for Turso/LibSQL
// databases used in CQRS event-sourcing workloads.
//
// It analyzes EXPLAIN QUERY PLAN output to detect full-table scans, tracks
// query patterns, and recommends or automatically creates indexes tailored
// to common CQRS access patterns (aggregate loads, event journal reads,
// projection cursors, and command audits).
//
// Quick start — analyze a slow query:
//
//	advisor := indexing.NewAdvisor(db)
//	recs, _ := advisor.AnalyzeQuery(ctx,
//	    "SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ?")
//	for _, r := range recs {
//	    fmt.Println(r.Explanation, r.Index.DDL())
//	}
//
// Quick start — auto-apply CQRS-optimized indexes:
//
//	auto := indexing.NewAutoIndexer(db)
//	auto.Enable()
//	_ = auto.ApplyRecommended(ctx)
//
// Quick start — apply Turso performance pragmas:
//
//	_ = indexing.ApplyOptimizations(ctx, db)
