package tursoengine

import (
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Option configures optional tursoengine capabilities at construction.
type Option func(*options)

type options struct {
	matViewSpecs []metaengine.MaterializedViewSpec
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

// withExperimentalViews appends the libSQL "views" experimental-feature flag
// to an embedded DSN (e.g. ":memory:?experimental=views"). Remote DSNs pass
// through unchanged: the driver ignores DSN params there, and the flag is a
// server-side setting (libsqld --experimental-views) — construction then
// fails with the server's own clear error if the feature is missing.
func withExperimentalViews(dsn string) string {
	if isRemoteDSN(dsn) || strings.Contains(dsn, "experimental=") {
		return dsn
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}

	return dsn + sep + "experimental=views"
}
