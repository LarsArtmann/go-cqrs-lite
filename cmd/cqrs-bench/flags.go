package main

import (
	"flag"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

// benchFlags holds the pointer values for the flags shared across every
// `cqrs-bench` subcommand. Populate with registerBenchFlags, then deref at
// the call site.
type benchFlags struct {
	backend      *string
	dsn          *string
	dir          *string
	profileName  *string
	codecName    *string
	format       *string
	output       *string
	payloadSize  *int
	payloadSizes *string
	durability   *string
	skipRawSink  *bool
	skipJourney  *bool
	skipQuery    *bool
	skipSnapshot *bool
	skipMixed    *bool
	progress     *time.Duration
	repeat       *int
}

// registerBenchFlags wires the flags every subcommand exposes (backend, dsn,
// dir, profile, codec, format, output, payload size/sizes, skip-* flags,
// repeat) onto the given FlagSet and returns their pointers. Centralising the
// declarations here keeps the three subcommands in lockstep when a new flag
// is added.
// newBenchFlagSet creates a flag set for the given subcommand name and
// registers the shared bench flags on it. Used by run, compare, and sweep.
func newBenchFlagSet(name string) (*flag.FlagSet, benchFlags) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)

	return fs, registerBenchFlags(fs)
}

func registerBenchFlags(fs *flag.FlagSet) benchFlags {
	return benchFlags{
		backend: fs.String(
			"backend",
			"memory",
			"Backend: memory, sqlite, pebble, postgres, duckdb, turso",
		),
		dsn:         fs.String("dsn", "", "Database connection string (sqlite, postgres, duckdb)"),
		dir:         fs.String("dir", "", "Database directory (pebble, duckdb)"),
		profileName: fs.String("profile", "dev", "Workload profile"),
		codecName:   fs.String("codec", "json", "Payload codec: json, cbor"),
		format:      fs.String("format", "text", "Output format: text, json, benchstat, manifest"),
		output:      fs.String("output", "", "Output file (default: stdout)"),
		payloadSize: fs.Int("payload-size", 256, "Payload size in bytes per event"),
		payloadSizes: fs.String(
			"payload-sizes",
			"",
			"Comma-separated payload sizes for a MIXED workload (e.g. 64,256,4096). Overrides --payload-size",
		),
		skipRawSink: fs.Bool("skip-raw-sink", false, "Skip raw prebuilt-event sink phase"),
		skipJourney: fs.Bool(
			"skip-journey",
			false,
			"Skip end-to-end publish→projection→query journey phase",
		),
		skipQuery:    fs.Bool("skip-query", false, "Skip typed query dispatch phase"),
		skipSnapshot: fs.Bool("skip-snapshot", false, "Skip snapshot/cache hit-rate phase"),
		skipMixed:    fs.Bool("skip-mixed", false, "Skip concurrent read-during-write phase"),
		progress: fs.Duration("progress", 5*time.Second,
			"Progress update interval to stderr (0 disables)"),
		durability: fs.String(
			"durability",
			"",
			"Durability tier: strict, normal, relaxed (default: normal)",
		),
		repeat: fs.Int("repeat", 0, "Run N times, report median (reduces ~20% variance)"),
	}
}

// loadProfileAndCodec resolves the workload profile and payload codec from
// their flag values, calling fatalf on an unknown profile name. Used by every
// subcommand that actually runs a benchmark.
func loadProfileAndCodec(profileName, codecName string) (benchkit.Profile, codec.Codec) {
	profile, ok := benchkit.ProfileByName(profileName)
	if !ok {
		fatalf("unknown profile: %s", profileName)
	}

	return profile, parseCodec(codecName)
}
