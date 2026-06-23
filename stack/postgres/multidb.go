package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// openSecondaryDB opens and configures a secondary Postgres database (for
// events, queries, or views when multi-DB mode is enabled).
func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres preset: open %q: %w", dsn, err)
	}

	if cfg.autoMigrate {
		ctx := context.Background()

		err = storage.PostgresInitSchema(ctx, db)
		if err != nil {
			_ = db.Close()

			return nil, fmt.Errorf("postgres preset: init schema on %q: %w", dsn, err)
		}
	}

	return db, nil
}

// openSecondaryBackend opens a secondary Postgres database, creates its
// backend, and returns both along with a closer. Shared by the event-DB and
// query-DB paths.
func openSecondaryBackend(
	dsn string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	secDB, err := openSecondaryDB(dsn, cfg)
	if err != nil {
		return nil, nil, err
	}

	secBackend, err := storage.NewSQLBackend(secDB)
	if err != nil {
		_ = secDB.Close()

		return nil, nil, fmt.Errorf("postgres preset: create backend for %q: %w", dsn, err)
	}

	closer := &multiCloser{closers: []io.Closer{secBackend, &funcCloser{fn: secDB.Close}}}

	return secBackend, closer, nil
}

// openEventStores opens a secondary database for the event-sourcing write
// model: the event store, snapshots, and checkpoints.
func openEventStores(
	dsn string,
	cfg config,
) ([]stack.Option, io.Closer, error) {
	secBackend, closer, err := openSecondaryBackend(dsn, cfg)
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
// stores.
func openQueryStores(
	dsn string,
	cfg config,
) ([]stack.Option, io.Closer, error) {
	secBackend, closer, err := openSecondaryBackend(dsn, cfg)
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
