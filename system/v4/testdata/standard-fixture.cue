// Fixtures for scripts/check-cue-config.sh: engine declarations evaluated
// against the `standard` preset. The rendered result is pinned by
// standard-topology.golden.yaml, and the instance triple by
// standard-instance-triple.json (shared with the Go suite and the NixOS eval
// test — the ADR-0165 cross-layer equivalence pin; the engine names
// primary+proj are load-bearing for that pin).
package cqrs

standard: engines: primary: {
	driver: "sqlite"
	dsn: "file:app.db"
	pragmas: ["journal_mode=wal"]
}

standard: engines: proj: {
	driver: "memory"
}
