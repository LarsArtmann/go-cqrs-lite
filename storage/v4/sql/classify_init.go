package sql

import (
	"errors"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

func init() { //nolint:gochecknoinits // package-wide registration of stdlib/driver error classifiers, must run before any store operation
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
	errorfamily.RegisterClassifiers(classifySQLiteError, classifyPostgresError, classifyMySQLError)
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

// classifyMySQLError classifies MySQL/MariaDB errors. The go-sql-driver/mysql
// driver exposes error codes via the *MySQLError.Number field (not a method),
// which cannot be matched by an interface without importing the driver. This
// classifier uses the driver's stable "Error NNNN:" message prefix instead.
//
// Error numbers:
//
//   - 1062 (ER_DUP_ENTRY) → Conflict (duplicate key, not retryable)
//   - 1205 (ER_LOCK_WAIT_TIMEOUT) → Transient (retryable)
//   - 1213 (ER_DEADLOCK) → Transient (retryable)
//   - 2003 (CR_CONN_HOST_ERROR) → Transient (retryable)
//   - 2006 (CR_SERVER_GONE_ERROR) → Transient (retryable)
//   - 2013 (CR_SERVER_LOST) → Transient (retryable)
func classifyMySQLError(err error) (errorfamily.Family, bool) {
	// Typed check via mysqlNumberError interface (forward-looking).
	if me, ok := errors.AsType[mysqlNumberError](err); ok {
		switch me.Number() {
		case mysqlDupNumber: // 1062
			return errorfamily.Conflict, true
		case 1205, 1213, 2003, 2006, 2013:
			return errorfamily.Transient, true
		}
		return errorfamily.Transient, false
	}

	// String-based fallback: the go-sql-driver/mysql error format is
	// "Error NNNN: <message>".
	msg := err.Error()

	switch {
	case strings.Contains(msg, "Error 1062"):
		return errorfamily.Conflict, true
	case strings.Contains(msg, "Error 1205"),
		strings.Contains(msg, "Error 1213"),
		strings.Contains(msg, "Error 2003"),
		strings.Contains(msg, "Error 2006"),
		strings.Contains(msg, "Error 2013"):
		return errorfamily.Transient, true
	default:
		return errorfamily.Transient, false
	}
}
