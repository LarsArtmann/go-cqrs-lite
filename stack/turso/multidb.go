package turso

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// openSecondaryBackend opens and configures a secondary Turso database,
// creates its backend, and returns both along with a closer that releases
// the backend and the *sql.DB. Shared by the event-DB and query-DB paths.
func openSecondaryBackend(
	dbPath string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	secDB, err := openSecondaryDB(dbPath, cfg)
	if err != nil {
		return nil, nil, err
	}

	secBackend, err := cqrsturso.NewBackend(secDB)
	if err != nil {
		_ = secDB.Close()

		return nil, nil, fmt.Errorf("turso: create backend for %q: %w", dbPath, err)
	}

	closer := &multiCloser{closers: []io.Closer{secBackend, &funcCloser{fn: secDB.Close}}}

	return secBackend, closer, nil
}

// openEventStores opens a secondary database for the event-sourcing write
// model: the event store, snapshots, and checkpoints. These three stores
// together serve event sourcing, so they share one database. Commands and
// queries are NOT placed here — see [openQueryStores].
func openEventStores(
	dbPath string,
	cfg config,
) ([]stack.Option, io.Closer, error) {
	secBackend, closer, err := openSecondaryBackend(dbPath, cfg)
	if err != nil {
		return nil, nil, err
	}

	opts := []stack.Option{
		stack.WithEventStore(secBackend.EventStore()),
	}

	if snapStore, snapErr := secBackend.SnapshotStore(); snapErr == nil {
		opts = append(opts, stack.WithSnapshotStore(snapStore))
	}

	if cpStore, cpErr := secBackend.CheckpointStore(); cpErr == nil {
		opts = append(opts, stack.WithCheckpointStore(cpStore))
	}

	return opts, closer, nil
}

// openQueryStores opens a secondary database for the command and query audit
// stores — the operational log of what was dispatched. Events, snapshots, and
// checkpoints are NOT placed here — see [openEventStores].
func openQueryStores(
	dbPath string,
	cfg config,
) ([]stack.Option, io.Closer, error) {
	secBackend, closer, err := openSecondaryBackend(dbPath, cfg)
	if err != nil {
		return nil, nil, err
	}

	var opts []stack.Option

	if cmdStore, cmdErr := secBackend.CommandStore(); cmdErr == nil {
		opts = append(opts, stack.WithCommandStore(cmdStore))
	}

	if qStore, qErr := secBackend.QueryStore(); qErr == nil {
		opts = append(opts, stack.WithQueryStore(qStore))
	}

	return opts, closer, nil
}

// openSecondaryDB opens and configures a secondary Turso database.
func openSecondaryDB(dbPath string, cfg config) (*sql.DB, error) {
	sqlDB, err := cqrsturso.Open(cqrsturso.DbPath(dbPath))
	if err != nil {
		return nil, fmt.Errorf("turso: open %q: %w", dbPath, err)
	}

	cqrsturso.ConfigurePool(sqlDB)

	if cfg.autoMigrate {
		ctx := context.Background()
		err := cqrsturso.InitSchema(ctx, sqlDB)
		if err != nil {
			_ = sqlDB.Close()

			return nil, fmt.Errorf("turso: init schema on %q: %w", dbPath, err)
		}
	}

	return sqlDB, nil
}
