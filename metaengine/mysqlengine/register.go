package mysqlengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func init() {
	metaengine.RegisterDriver(
		"mysql",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			return New(cfg.DSN) //nolint:contextcheck // constructor doesn't take ctx
		},
	)
}
