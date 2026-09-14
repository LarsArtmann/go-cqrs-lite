package mysql

import (
	"context"
	"database/sql"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func openSecondaryDB(dsn string, cfg config) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "mysql_preset.open_secondary",
			"open MySQL secondary DB")
	}

	if cfg.AutoMigrate {
		if err := storage.MySQLInitSchema(context.Background(), db); err != nil {
			_ = db.Close()

			return nil, errorfamily.WrapInfrastructure(err, "mysql_preset.secondary_init_schema",
				"initialize MySQL schema on secondary DB")
		}
	}

	return db, nil
}

func openSecondaryBackend(
	dsn string,
	cfg config,
) (*storage.SQLBackend, io.Closer, error) {
	backend, closer, err := sqlopt.NewSecondaryBackend(dsn,
		func() (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewMySQLBackend,
		"mysql_preset.create_secondary_backend")
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(
			err,
			"mysql_preset.create_secondary_backend",
			"create secondary MySQL backend",
		)
	}

	return backend, closer, nil
}
