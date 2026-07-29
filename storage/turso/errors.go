package turso

import (
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrMemorySync is returned when trying to sync an in-memory Turso database.
var ErrMemorySync = errorfamily.NewRejection(
	"turso.memory_sync",
	"turso: sync requires a file path for dbPath",
)

// ErrQuotaExceeded represents a Turso plan-limit error. Turso returns HTTP 403
// with a body like:
//
//	{"error":"Operation was blocked: SQL read operations are forbidden
//	(reads are blocked, do you need to upgrade your plan?)"}
//
// On the free tier, hitting ANY limit (rows read, rows written, storage, or
// sync bandwidth) blocks the entire account. This is NOT a data problem — the
// local database is intact. Retrying immediately burns more quota; the caller
// should back off exponentially or switch to a local-only backend.
//
// Use [IsQuotaExceeded] to detect this class of error on any wrapped error.
var ErrQuotaExceeded = errorfamily.NewRejection(
	"turso.quota_exceeded",
	"turso: account quota exceeded — upgrade plan or reduce usage",
)

// IsQuotaExceeded reports whether err is (or wraps) a Turso quota/plan-limit
// error. Turso returns HTTP 403 with distinctive body strings when any plan
// limit is hit. This function walks the full error message (including all
// wrapped causes) via err.Error().
//
// This is distinct from local corruption ([isTursoSyncLocalCorruption] in
// consumers): quota errors mean the data is fine but the account is throttled,
// while corruption means the local file is damaged and needs rebuilding.
func IsQuotaExceeded(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "Operation was blocked") ||
		strings.Contains(msg, "SQL read operations are forbidden") ||
		strings.Contains(msg, "SQL write operations are forbidden") ||
		strings.Contains(msg, "reads are blocked") ||
		strings.Contains(msg, "writes are blocked") ||
		strings.Contains(msg, "do you need to upgrade your plan")
}

// wrapInfraOrOK returns nil when err is nil, otherwise wraps err as an
// infrastructure error with the given code and message. Collapses the
// repeated "if err != nil { return WrapInfrastructure(...) }; return nil"
// boilerplate in SyncDB methods — the unique code stays a parameter.
//
// If the error is a Turso quota/plan-limit ([IsQuotaExceeded]), it is wrapped
// as a Rejection (not Infrastructure) so callers can distinguish "throttled,
// back off" from "transient, retry soon" via [errors.Is](err, ErrQuotaExceeded)
// or [IsQuotaExceeded](err).
func wrapInfraOrOK(err error, code, msg string) error {
	if err == nil {
		return nil
	}

	if IsQuotaExceeded(err) {
		return errorfamily.WrapRejection(err, "turso.quota_exceeded", msg)
	}

	return errorfamily.WrapInfrastructure(err, code, msg)
}
