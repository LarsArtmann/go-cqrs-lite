package tursoengine

import (
	"errors"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Option configures optional tursoengine capabilities at construction.
type Option func(*options)

type options struct {
	matViewSpecs []metaengine.MaterializedViewSpec
	encryption   *encryptionConfig

	// knownGroupedViewBug acknowledges the upstream grouped-view defect
	// (WithKnownGroupedViewBug): grouped materialized views diverge from the
	// second transaction on and collapse at ~27k rows on turso-go ≤ v0.8.x.
	knownGroupedViewBug bool
}

// ErrGroupedViewBugRefused is returned by New when a materialized view spec
// declares GroupBy without WithKnownGroupedViewBug: grouped views on turso-go
// silently return WRONG RESULTS from the second transaction on (upstream
// defects A+B, see docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md;
// the commit-abort half, defect C, is reported at turso PR #8257). Scalar
// views (no GroupBy) are exact and unaffected.
var ErrGroupedViewBugRefused = errors.New(
	"tursoengine: grouped materialized view refused: upstream turso-go grouped views silently diverge " +
		"(wrong results from the second transaction on, collapse at ~27k rows); " +
		"pass WithKnownGroupedViewBug() to acknowledge and opt in, or drop GroupBy (scalar views are exact)",
)

// WithKnownGroupedViewBug opts in to grouped materialized view specs despite
// the upstream silent-wrong-results defect (see ErrGroupedViewBugRefused).
// Only for deployments that have verified their grouped results independently
// or accept the risk on small data — the defect's onset boundary is
// scan-activity-sensitive (~24k-27k rows observed).
func WithKnownGroupedViewBug() Option {
	return func(o *options) {
		o.knownGroupedViewBug = true
	}
}

// WithMaterializedViews registers operator-declared materialized views
// (metaengine.MaterializedViewSpec). The engine enables the required
// "views" experimental feature on embedded DSNs automatically and creates
// each view (CREATE MATERIALIZED VIEW IF NOT EXISTS) at construction —
// failing loudly when the server cannot maintain materialized views.
func WithMaterializedViews(specs []metaengine.MaterializedViewSpec) Option {
	return func(o *options) {
		o.matViewSpecs = append(o.matViewSpecs, specs...)
	}
}

// withExperimentalViews appends the "views" experimental-feature flag to an
// embedded DSN (e.g. ":memory:?experimental=views"). An existing
// experimental list is MERGED (comma-separated), so a DSN already carrying
// e.g. "experimental=encryption" gains "encryption,views" instead of losing
// the views flag. Remote DSNs pass through unchanged: the driver ignores DSN
// params there, and the flag is a server-side setting
// (libsqld --experimental-views) — construction then fails with the server's
// own clear error if the feature is missing.
func withExperimentalViews(dsn string) string {
	return withExperimentalToken(dsn, "views")
}
