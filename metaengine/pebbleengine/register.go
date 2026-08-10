package pebbleengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func init() {
	metaengine.RegisterDriver(
		"pebble",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			return NewPebbleEngine(cfg.DSN)
		},
	)
}
