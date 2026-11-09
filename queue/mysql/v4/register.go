package mysql

import (
	"context"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// errDSNRequired is the driver-factory rejection when no DSN is configured.
var errDSNRequired = errors.New("queue-mysql: DSN required")

// RegisterDriver registers the queue-mysql engine under its driver name so
// operator config can pick it (the database/sql pattern; ADR-0142 T09).
// Import for side effects: `_ "github.com/larsartmann/go-cqrs-lite/queue/mysql/v4"`.
//
// art-dupl:accept each dep-isolated engine module needs its own init() calling metaengine.RegisterDriver (the database/sql registration pattern, AGENTS contract 19)
func init() {
	metaengine.RegisterDriver(
		"queue-mysql",
		func(ctx context.Context, cfg metaengine.DriverConfig) (metaengine.Engine, error) {
			if cfg.DSN == "" {
				return nil, errDSNRequired
			}

			eng, err := NewEngine(ctx, cfg.DSN)
			if err != nil {
				return nil, err
			}

			if err := eng.PingContext(ctx); err != nil {
				_ = eng.Close()

				return nil, fmt.Errorf("queue-mysql: ping: %w", err)
			}

			return eng, nil
		},
	)
}
