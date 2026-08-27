package system

import "errors"

// Sentinel errors for the system module. Callers can use errors.Is to check
// for specific failure conditions.
var (
	ErrAlreadyStarted         = errors.New("system: already started")
	ErrCacheCapacityInvalid   = errors.New("system: cache capacity must be positive")
	ErrCommandTypeMismatch    = errors.New("system: command type mismatch")
	ErrDeciderTypeMismatch    = errors.New("system: decider type mismatch")
	ErrDuplicateInstanceRole  = errors.New("system: duplicate dedicated instance role")
	ErrDurabilityConflict     = errors.New("system: conflicting durability tiers for engine")
	ErrEventStoreMissing      = errors.New("system: no event store")
	ErrJournalMissing         = errors.New("system: store does not implement event.Journal")
	ErrNoDecider              = errors.New("system: no decider registered for stream type")
	ErrNoProjectionHost       = errors.New("system: no projection host configured")
	ErrNoProjections          = errors.New("system: no projections configured")
	ErrNotSnapshotBackend     = errors.New("system: engine does not implement SnapshotBackend")
	ErrNotStreamLogBackend    = errors.New("system: engine does not implement StreamLogBackend")
	ErrQueryResultMismatch    = errors.New("system: query result type mismatch")
	ErrQueryTypeMismatch      = errors.New("system: query type mismatch")
	ErrSeekableJournalMissing = errors.New("system: store does not implement event.SeekableJournal")
	ErrShutdownDependencyInvalid = errors.New("system: invalid shutdown dependency")
	ErrSystemStopped          = errors.New("system: already stopped")
	ErrUnknownBusDriver       = errors.New("system: unknown bus driver")
	ErrUnknownEngine          = errors.New("system: unknown engine")
	ErrUnsupportedValueType   = errors.New("system: unsupported value type")
)
