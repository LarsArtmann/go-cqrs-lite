// Fixture for the schema-only `deployment` entry point: explicit instances
// pass through VERBATIM (the standard filler is opt-in through the
// `standard` entry, never implicit here). Pinned by
// deployment-explicit.golden.yaml via scripts/check-cue-config.sh.
package cqrs

deployment: engines: primary: {
	driver: "sqlite"
	dsn: "file:app.db"
}

deployment: instances: [
	{
		role: "source-of-truth"
		engine: "primary"
		durability: "strict"
	},
	{
		role: "projections"
		engines: ["primary"]
	},
]
