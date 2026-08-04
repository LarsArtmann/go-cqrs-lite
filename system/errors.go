package system

import "errors"

// Sentinel errors for the system module. Callers can use errors.Is to check
// for specific failure conditions.
var (
	ErrAlreadyStarted         = errors.New("system: already started")
	ErrCacheCapacityInvalid   = errors.New("system: cache capacity must be positive")
	ErrCommandTypeMismatch    = errors.New("system: command type mismatch")
	ErrDeciderTypeMismatch    = errors.New("system: decider type mismatch")
	ErrEventStoreMissing      = errors.New("system: no event store")
	ErrJournalMissing         = errors.New("system: store does not implement event.Journal")
	ErrNoDecider              = errors.New("system: no decider registered for stream type")
	ErrNotStreamLogBackend    = errors.New("system: engine does not implement StreamLogBackend")
	ErrQueryResultMismatch    = errors.New("system: query result type mismatch")
	ErrQueryTypeMismatch      = errors.New("system: query type mismatch")
	ErrSeekableJournalMissing = errors.New("system: store does not implement event.SeekableJournal")
	ErrUnknownBusDriver       = errors.New("system: unknown bus driver")
	ErrUnknownDriver          = errors.New("system: unknown driver")
	ErrUnknownEngine          = errors.New("system: unknown engine")
	ErrUnsupportedValueType   = errors.New("system: unsupported value type")
)
