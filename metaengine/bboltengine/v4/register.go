package bboltengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// art-dupl:accept database/sql-style driver self-registration: init() must live in each dep-isolated engine module
func init() {
	metaengine.RegisterDriver(
		"bbolt",
		func(_ context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			opts, err := tierToOptions(cfg.Durability)
			if err != nil {
				return nil, err
			}

			return NewBboltEngine(
				cfg.DSN,
				opts...) //nolint:contextcheck,wrapcheck // no ctx; factory passthrough
		},
	)
}
