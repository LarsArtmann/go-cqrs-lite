// Package graphadapter the bridge from graph.GraphDriver to metaengine.Engine.
//
// # Limitation: flat node identity
//
// GraphAddEdge synthesizes NodeRefs with a fixed label ("entity") and key
// prop ("id"), so every node lands in ONE namespace — distinct node types
// (User vs Task) are not distinguishable in adapter-built graphs, and
// fmt.Sprint stringifies non-string endpoint values. For label-rich reads,
// bypass the adapter: Driver() returns the underlying graph.MemoryDriver
// (Traverse/Neighbors/ShortestPath over properly labeled NodeRefs). The
// engine-backed Graph ADT path (sqlite, pg, dgraph, ... via
// metaengine.Query Edge folds) does not route through this adapter and is
// unaffected. Extending Edge with node labels is a Proposed v5 decision
// (ADR-0156 "Graph edge labels at v5", Proposed).
//
// # Experimental
//
// Part of the metaengine family — the cost-based storage planner and its
// engine backends are experimental: the API may change between minor
// versions (FEATURES.md, "Metaengine: EXPERIMENTAL").
package graphadapter
