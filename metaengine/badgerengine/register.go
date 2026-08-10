package badgerengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func init() {
	metaengine.RegisterDriver("badger", func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
		return NewBadgerEngine(cfg.DSN)
	})
}
