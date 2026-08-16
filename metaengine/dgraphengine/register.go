package dgraphengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept database/sql-style driver self-registration: init() must live in each dep-isolated engine module
func init() {
	metaengine.RegisterDriver(
		"dgraph",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			return New(cfg.DSN) //nolint:contextcheck // constructor doesn't take ctx
		},
	)
}
