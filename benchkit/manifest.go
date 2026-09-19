package benchkit

import "io"

// SuiteManifest records the complete context for a benchmark result:
// the config that produced it, the environment it ran in, and the result.
// This enables reproducibility: a manifest can be archived alongside a
// release to document exactly what was measured and under what conditions.
type SuiteManifest struct {
	SchemaVersion string      `json:"schemaVersion"`
	Config        Config      `json:"config"`
	Environment   Environment `json:"environment"`
	Result        *Result     `json:"result"`

	// Runs holds every repeat run in execution order when the manifest was
	// built from a multi-run benchmark via [NewManifestRepeated]. It is
	// deliberately opt-in: N repeats multiply the file size N-fold, and the
	// median in Result is what single-run consumers read. Raw per-run data
	// is what lets a later analysis recompute dispersion or feed benchstat
	// without re-running the benchmark.
	Runs []*Result `json:"runs,omitempty"`
}

// NewManifest creates a SuiteManifest from a Config and Result.
func NewManifest(config Config, result *Result) SuiteManifest {
	return SuiteManifest{
		SchemaVersion: SchemaVersion,
		Config:        config,
		Environment:   result.Environment,
		Result:        result,
	}
}

// WriteManifest serializes a SuiteManifest as indented JSON.
func WriteManifest(w io.Writer, config Config, result *Result) error {
	return writeJSONAny(w, NewManifest(config, result))
}

// NewManifestRepeated creates a SuiteManifest from a multi-run benchmark:
// Result is the median run, Runs every run in execution order. Use it when
// the manifest should carry the raw per-run data (see [SuiteManifest.Runs]).
func NewManifestRepeated(config Config, repeated *RepeatedResult) SuiteManifest {
	median := repeated.Median
	if median == nil && len(repeated.Runs) > 0 {
		median = repeated.Runs[0]
	}

	return SuiteManifest{
		SchemaVersion: SchemaVersion,
		Config:        config,
		Environment:   median.Environment,
		Result:        median,
		Runs:          repeated.Runs,
	}
}

// WriteManifestRepeated serializes a multi-run SuiteManifest (median + every
// repeat run) as indented JSON. Size grows linearly with the repeat count —
// that is the point: the per-run data is what later dispersion analysis
// consumes. Single-run code should use [WriteManifest].
func WriteManifestRepeated(w io.Writer, config Config, repeated *RepeatedResult) error {
	return writeJSONAny(w, NewManifestRepeated(config, repeated))
}
