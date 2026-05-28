package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/saga"
)

// SQLSagaStore persists saga state in a SQL database.
type SQLSagaStore struct {
	sqlBase
}

// NewSQLSagaStore creates a new SQL-backed saga store using PostgreSQL dialect.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLSagaStore(db *sql.DB) (*SQLSagaStore, error) {
	return newSQLSagaStoreWithDialect(db, PostgresDialect{})
}

// NewSQLiteSagaStore creates a new SQLite-backed saga store.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLiteSagaStore(db *sql.DB) (*SQLSagaStore, error) {
	return newSQLSagaStoreWithDialect(db, SQLiteDialect{})
}

// NewSQLSagaStoreWithDialect creates a new SQL-backed saga store with a custom dialect.
// This enables consumers to use any SQL backend by implementing the Dialect interface.
// The *sql.DB is borrowed, not owned — the caller is responsible for closing it.
// Returns an error if db is nil.
func NewSQLSagaStoreWithDialect(db *sql.DB, d Dialect) (*SQLSagaStore, error) {
	return newSQLSagaStoreWithDialect(db, d)
}

func newSQLSagaStoreWithDialect(db *sql.DB, d Dialect) (*SQLSagaStore, error) {
	base, err := newSQLBase(db, d)
	if err != nil {
		return nil, err
	}

	return &SQLSagaStore{sqlBase: base}, nil
}

// SagaSchema returns the SQL DDL for creating the sagas table.
func SagaSchema() string { return PostgresDialect{}.SagaSchema() }

// SQLiteSagaSchema returns the SQL DDL for creating the sagas table (SQLite variant).
func SQLiteSagaSchema() string { return SQLiteDialect{}.SagaSchema() }

// Save persists a saga state with UPSERT semantics.
func (s *SQLSagaStore) Save(ctx context.Context, state *saga.State) error {
	if state == nil {
		return event.WrapInfrastructure(ErrNilDB, "storage.nil_saga_state",
			"saga state is nil")
	}

	p1, p2, p3, p4, p5, p6, p7 := s.dialect.Placeholder(1), s.dialect.Placeholder(2),
		s.dialect.Placeholder(3), s.dialect.Placeholder(4),
		s.dialect.Placeholder(5), s.dialect.Placeholder(6),
		s.dialect.Placeholder(7)

	query := fmt.Sprintf(
		`INSERT INTO `+tableSagas+` (id, saga_type, status, current_step, err_msg, created_at, updated_at)
		VALUES (%s, %s, %s, %s, %s, %s, %s)
		ON CONFLICT (id)
		DO UPDATE SET saga_type = EXCLUDED.saga_type, status = EXCLUDED.status,
		              current_step = EXCLUDED.current_step, err_msg = EXCLUDED.err_msg,
		              updated_at = EXCLUDED.updated_at`,
		p1,
		p2,
		p3,
		p4,
		p5,
		p6,
		p7,
	)

	_, err := s.db.ExecContext(
		ctx,
		query,
		state.ID.String(),
		state.SagaType,
		string(state.Status),
		state.CurrentStep,
		state.ErrMsg,
		s.dialect.FormatTime(state.CreatedAt),
		s.dialect.FormatTime(state.UpdatedAt),
	)
	if err != nil {
		return event.WrapInfrastructure(err, "storage.save_saga",
			"save saga "+state.ID.String())
	}

	return nil
}

// Load retrieves a saga state by ID.
func (s *SQLSagaStore) Load(ctx context.Context, id id.AggregateID) (*saga.State, error) {
	p1 := s.dialect.Placeholder(1)

	query := fmt.Sprintf(
		`SELECT saga_type, status, current_step, err_msg, created_at, updated_at
		FROM `+tableSagas+` WHERE id = %s`,
		p1,
	)

	row := s.db.QueryRowContext(ctx, query, id.String())

	return s.scanState(row, id)
}

// LoadAllRunning returns all saga states that are currently running or compensating.
func (s *SQLSagaStore) LoadAllRunning(ctx context.Context) ([]*saga.State, error) {
	query := `SELECT id, saga_type, status, current_step, err_msg, created_at, updated_at
		FROM ` + tableSagas + ` WHERE status = ` + s.dialect.Placeholder(1) + ` OR status = ` + s.dialect.Placeholder(2)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		string(saga.StatusRunning),
		string(saga.StatusCompensating),
	)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.query_running_sagas",
			"query running sagas")
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanStates(rows)
}

func (s *SQLSagaStore) scanState(row *sql.Row, sagaID id.AggregateID) (*saga.State, error) {
	var (
		sagaType    string
		status      string
		currentStep int
		errMsg      string
	)

	createdAtDest := s.dialect.ScanTimeDest()
	updatedAtDest := s.dialect.ScanTimeDest()

	err := row.Scan(&sagaType, &status, &currentStep, &errMsg, createdAtDest, updatedAtDest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, event.WrapRejection(saga.ErrSagaNotFound, "storage.saga_not_found",
				"saga "+sagaID.String()+" not found")
		}

		return nil, event.WrapInfrastructure(err, "storage.scan_saga",
			"scan saga "+sagaID.String())
	}

	createdAt, err := s.dialect.ParseTime(createdAtDest)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.parse_saga_created_at",
			"parse created_at for saga "+sagaID.String())
	}

	updatedAt, err := s.dialect.ParseTime(updatedAtDest)
	if err != nil {
		return nil, event.WrapCorruption(err, "storage.parse_saga_updated_at",
			"parse updated_at for saga "+sagaID.String())
	}

	return &saga.State{
		ID:          sagaID,
		SagaType:    sagaType,
		Status:      saga.Status(status),
		CurrentStep: currentStep,
		ErrMsg:      errMsg,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *SQLSagaStore) scanStates(rows *sql.Rows) ([]*saga.State, error) {
	var states []*saga.State

	for rows.Next() {
		var (
			sagaID      string
			sagaType    string
			status      string
			currentStep int
			errMsg      string
		)

		createdAtDest := s.dialect.ScanTimeDest()
		updatedAtDest := s.dialect.ScanTimeDest()

		err := rows.Scan(
			&sagaID,
			&sagaType,
			&status,
			&currentStep,
			&errMsg,
			createdAtDest,
			updatedAtDest,
		)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "storage.scan_running_saga",
				"scan running saga")
		}

		idParsed, err := id.ParseAggregateID(sagaID)
		if err != nil {
			return nil, event.WrapCorruption(err, "storage.parse_saga_id",
				"parse saga ID "+sagaID)
		}

		createdAt, err := s.dialect.ParseTime(createdAtDest)
		if err != nil {
			return nil, event.WrapCorruption(err, "storage.parse_running_created_at",
				"parse created_at for saga "+sagaID)
		}

		updatedAt, err := s.dialect.ParseTime(updatedAtDest)
		if err != nil {
			return nil, event.WrapCorruption(err, "storage.parse_running_updated_at",
				"parse updated_at for saga "+sagaID)
		}

		states = append(states, &saga.State{
			ID:          idParsed,
			SagaType:    sagaType,
			Status:      saga.Status(status),
			CurrentStep: currentStep,
			ErrMsg:      errMsg,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, event.WrapInfrastructure(err, "storage.iterate_sagas",
			"iterate running sagas")
	}

	return states, nil
}

var _ saga.Store = (*SQLSagaStore)(nil)
