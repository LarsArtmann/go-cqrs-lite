package sql

import (
	"errors"

	errorfamily "github.com/larsartmann/go-error-family"
)

func init() {
	// Register stdlib error classifications so database/sql and context
	// errors classify correctly throughout the storage layer:
	//   sql.ErrNoRows     → Rejection (caller's concern, not a system fault)
	//   context.Canceled  → Rejection (caller abandoned the operation)
	//   sql.ErrConnDone   → Transient (retry on a fresh connection)
	//   context.DeadlineExceeded → Transient (retryable)
	errorfamily.RegisterStdlibDefaults(errorfamily.DefaultRegistry)

	// Register classifiers for database driver errors that cannot be matched
	// by sentinel identity (each error is a fresh instance). These use the
	// same interface-based detection as IsDuplicateKeyError, so no additional
	// driver dependencies are introduced.
	errorfamily.RegisterClassifiers(classifySQLiteError, classifyPostgresError)
}

// classifySQLiteError classifies modernc.org/sqlite errors via the
// sqliteCodeError interface (Code() int). SQLite result codes:
//
//   - 5  (SQLITE_BUSY)     → Transient (retryable: another connection holds a lock)
//   - 6  (SQLITE_LOCKED)   → Transient (retryable: table is locked)
//   - 19 (SQLITE_CONSTRAINT) → Conflict (constraint violation, not retryable)
func classifySQLiteError(err error) (errorfamily.Family, bool) {
	ce, ok := errors.AsType[sqliteCodeError](err)
	if !ok {
		return errorfamily.Transient, false
	}

	switch ce.Code() {
	case 5, 6: // SQLITE_BUSY, SQLITE_LOCKED
		return errorfamily.Transient, true
	case 19: // SQLITE_CONSTRAINT
		return errorfamily.Conflict, true
	default:
		return errorfamily.Transient, false
	}
}

// classifyPostgresError classifies PostgreSQL errors via the pgCodeError
// interface (Code() string — SQLSTATE). Uses the SQLSTATE class prefix
// (first 2 chars) for broad coverage:
//
//   - Class 23 (integrity constraint violation) → Conflict
//   - Class 40 (transaction rollback)           → Transient
//   - Class 53 (insufficient resources)         → Transient
//   - Class 57 (operator intervention)          → Transient
//   - Class 58 (system error)                   → Transient
func classifyPostgresError(err error) (errorfamily.Family, bool) {
	ce, ok := errors.AsType[pgCodeError](err)
	if !ok {
		return errorfamily.Transient, false
	}

	code := ce.Code()
	if len(code) < 2 {
		return errorfamily.Transient, false
	}

	switch code[:2] {
	case "23": // integrity constraint violation
		return errorfamily.Conflict, true
	case "40", "53", "57", "58": // transient classes
		return errorfamily.Transient, true
	default:
		return errorfamily.Transient, false
	}
}
