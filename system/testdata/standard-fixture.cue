// Fixture for scripts/check-cue-config.sh: an engine declaration evaluated
// against the `standard` preset. The rendered result is pinned by
// standard-topology.golden.yaml.
package cqrs

standard: engines: primary: {
	driver: "sqlite"
	dsn: "file:app.db"
	pragmas: ["journal_mode=wal"]
}
