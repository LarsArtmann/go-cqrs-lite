package bboltengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func init() {
	metaengine.RegisterDriver(
		"bbolt",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			return NewBboltEngine(cfg.DSN) //nolint:contextcheck // constructor doesn't take ctx
		},
	)
}
