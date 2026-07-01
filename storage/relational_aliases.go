package storage

import "github.com/larsartmann/go-cqrs-lite/storage/v3/relational"

// Relational sub-package re-exports.
// Consumers importing storage/ can use these types unchanged.
// New consumers should prefer importing storage/relational directly.
var (
	NewRelationalProjection = relational.NewRelationalProjection //nolint:gochecknoglobals // backward-compat re-export
	NewRelationalStore      = relational.NewRelationalStore      //nolint:gochecknoglobals // backward-compat re-export
)

type (
	RelationalProjection       = relational.RelationalProjection
	RelationalProjectionOption = relational.RelationalProjectionOption
	RelationalSchema           = relational.RelationalSchema
	RelationalTable            = relational.RelationalTable
	RelationalColumn           = relational.RelationalColumn
	RelationalStore            = relational.RelationalStore
	RelationalHandler          = relational.RelationalHandler
	Row                        = relational.Row
	ProjectionSink             = relational.ProjectionSink
)

// WithoutRelationalAutoMigrate is re-exported for backward compatibility.
func WithoutRelationalAutoMigrate() RelationalProjectionOption {
	return relational.WithoutRelationalAutoMigrate()
}
