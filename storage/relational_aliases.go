package storage

import "github.com/larsartmann/go-cqrs-lite/storage/v4/relational"

// Relational sub-package re-exports.
// Consumers importing storage/ can use these types unchanged.
// New consumers should prefer importing storage/relational directly.
var (
	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	NewRelationalProjection = relational.NewRelationalProjection //nolint:gochecknoglobals // backward-compat re-export

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	NewRelationalStore = relational.NewRelationalStore //nolint:gochecknoglobals // backward-compat re-export
)

type (
	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	RelationalProjection = relational.RelationalProjection

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	RelationalProjectionOption = relational.RelationalProjectionOption

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	RelationalSchema = relational.RelationalSchema

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	RelationalTable = relational.RelationalTable

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	RelationalColumn = relational.RelationalColumn

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	RelationalStore = relational.RelationalStore

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	RelationalHandler = relational.RelationalHandler

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	Row = relational.Row

	// Deprecated: re-export of the storage/relational tier, removed at
	// v5 (ADR-0123): metaengine engines with layout planning.
	ProjectionSink = relational.ProjectionSink
)

// WithoutRelationalAutoMigrate is re-exported for backward compatibility.
//
// Deprecated: re-export of the storage/relational tier, removed at
// v5 (ADR-0123): metaengine engines with layout planning.
func WithoutRelationalAutoMigrate() RelationalProjectionOption {
	return relational.WithoutRelationalAutoMigrate()
}
