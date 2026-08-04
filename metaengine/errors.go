package metaengine

import (
	"errors"
	"fmt"
)

// Package-level sentinel errors. Dynamic call sites wrap these with engine
// names, query names, or type information via fmt.Errorf("...: %w", sentinel),
// preserving errors.Is matching for consumers while satisfying err113.

var (
	// Plan-time validation errors.
	errNoEngine     = errors.New("metaengine.Plan: at least one engine required")
	errNoQuery      = errors.New("metaengine.Plan: at least one query required")
	errNotQueryMeta = errors.New(
		"query does not implement queryMeta — pass a metaengine.Query[Q,R]",
	)
	errDuplicateQuery   = errors.New("metaengine.Plan: duplicate query name")
	errADTNotSupported  = errors.New("no engine supports it")
	errCannotInferADT   = errors.New("cannot infer ADT: no active folds (only skips)")
	errAmbiguousKey     = errors.New("ambiguous key: multiple fields of matching type")
	errNoKeyField       = errors.New("no field of matching type in event")
	errInvalidEventType = errors.New("handler first param must be")
	errEmptyField       = errors.New("declarative field has empty column name")

	// Plan-time validation errors for query metadata.
	errInvalidADT         = errors.New("query has invalid ADT")
	errInvalidReadPattern = errors.New("query has invalid ReadPattern")
	errInvalidFoldKind    = errors.New("query fold has invalid FoldKind")

	// Dispatch-time errors.
	errNoQueryForInputType = errors.New("no query declared for input type")
	errUnsupportedPattern  = errors.New("unsupported read pattern")
	errUnknownFoldKind     = errors.New("unknown fold kind")
	errExecuteTypeMismatch = errors.New(
		"metaengine.ExecuteTyped: result type does not match expected",
	)
	errKeyTypeMismatch = errors.New(
		"metaengine.Execute: input struct has no field matching the declared key type",
	)

	// Engine capability errors. Wrapped with the offending engine's name at the
	// call site, e.g. fmt.Errorf("%w: engine %s", errUnsupportedMapReads, name).
	errUnsupportedMapReads     = errors.New("engine does not support Map reads")
	errUnsupportedSetReads     = errors.New("engine does not support Set reads")
	errUnsupportedCounterReads = errors.New("engine does not support Counter reads")
	errUnsupportedGraphReads   = errors.New("engine does not support Graph reads")
	errUnsupportedMultiReads   = errors.New("engine does not support Multimap reads")
	errUnsupportedLogReads     = errors.New("engine does not support Log reads")
	errUnsupportedScanReads    = errors.New("engine does not support Scan reads")

	errUnsupportedMapOps      = errors.New("engine does not support Map operations")
	errUnsupportedCounterOps  = errors.New("engine does not support Counter operations")
	errUnsupportedGraphOps    = errors.New("engine does not support Graph operations")
	errUnsupportedSetOps      = errors.New("engine does not support Set operations")
	errUnsupportedMultimapOps = errors.New("engine does not support Multimap operations")
	errUnsupportedLogOps      = errors.New("engine does not support Log operations")

	errUnsupportedVectorReads = errors.New("engine does not support Vector reads")
	errUnsupportedSearchReads = errors.New("engine does not support Search reads")
	errUnsupportedVectorOps   = errors.New("engine does not support Vector operations")
	errUnsupportedSearchOps   = errors.New("engine does not support Search operations")

	errUnsupportedSpatialReads = errors.New("engine does not support Spatial reads")
	errUnsupportedSpatialOps   = errors.New("engine does not support Spatial operations")

	// Verify / consistency errors.
	errNoEventLog = errors.New(
		"metaengine.Verify: no event log attached — call WithEventLog first",
	)
	errNoQueryDecls            = errors.New("metaengine.Verify: no query declarations stored")
	errCollectionCountMismatch = errors.New("metaengine.Verify: collection count mismatch")
	errVerifyDrift             = errors.New("metaengine.Verify: collection row-count drift")

	// SwapEngine error.
	errSwapEngineNotFound = errors.New("metaengine.SwapEngine: engine not found")

	// SSE error.
	errSSENoFlusher = errors.New("metaengine.ServeSSE: response writer does not support flushing")

	// Coalescer error.
	errCoalescerTypeMismatch = errors.New("coalescer: unexpected result type")
)

// unsupportedEngine wraps a capability sentinel with the offending engine name.
func unsupportedEngine(base error, engineName string) error {
	return fmt.Errorf("%w: engine %s", base, engineName)
}

// ApplyError provides structured context for fold application failures.
// It wraps the underlying cause with the query name, event type, and fold kind,
// making debugging easier without grepping log strings.
type ApplyError struct {
	Query     string
	EventType string
	FoldKind  FoldKind
	Cause     error
}

func (e *ApplyError) Error() string {
	return fmt.Sprintf("query %q fold for %s (%s): %v",
		e.Query, e.EventType, e.FoldKind, e.Cause)
}

func (e *ApplyError) Unwrap() error { return e.Cause }

// --- Exported sentinel errors for consumers ---
//
// These allow consumers to use errors.Is for type-safe error matching instead
// of string matching. Each maps to an internal sentinel already in use.

var (
	// ErrNotFound is returned by ExecuteTyped when a point lookup finds no
	// value for the key. TypedReader.Get signals not-found via its bool
	// return (idiomatic Go map-lookup pattern) and does NOT return this error.
	ErrNotFound = errors.New("metaengine: key not found")

	// ErrAmbiguousKey is returned at Plan time when multiple fields of a struct
	// match the declared key type, making key extraction ambiguous.
	ErrAmbiguousKey = errAmbiguousKey

	// ErrUnsupportedADT is returned when no registered engine supports the
	// declared ADT for a query.
	ErrUnsupportedADT = errADTNotSupported

	// ErrLayoutConflict is returned when ApplyLayout or Plan detects a
	// conflicting table layout for the same collection.
	ErrLayoutConflict = errors.New("metaengine: conflicting layout plan")

	// ErrPoisoned is returned when a collection was poisoned by a fold panic.
	// Once poisoned, the collection refuses reads until the store is recreated.
	ErrPoisoned = errors.New("metaengine: collection poisoned by fold panic")

	// ErrNoQueryForInputType is returned when no registered query matches the
	// input struct type passed to Execute/ExecuteCtx.
	ErrNoQueryForInputType = errNoQueryForInputType

	// ErrUnsupportedPattern is returned when the engine does not support the
	// query's read pattern.
	ErrUnsupportedPattern = errUnsupportedPattern

	// ErrUnknownFoldKind is returned when a fold has an unrecognized FoldKind.
	ErrUnknownFoldKind = errUnknownFoldKind

	// ErrExecuteTypeMismatch is returned by ExecuteTyped when the result type
	// does not match the expected type parameter.
	ErrExecuteTypeMismatch = errExecuteTypeMismatch

	// ErrKeyTypeMismatch is returned by Execute/ExecuteTyped when the input
	// struct has no field matching the query's declared key type. This catches
	// a common footgun: passing the wrong struct to a point-lookup query.
	ErrKeyTypeMismatch = errKeyTypeMismatch

	// ErrDuplicateEvent is returned when ApplyIdempotent detects a duplicate
	// event ID. The event is silently skipped (no error returned to caller);
	// this sentinel is available for consumers building their own dedup layers.
	ErrDuplicateEvent = errors.New("metaengine: duplicate event (idempotent skip)")

	// ErrVersionConflict is returned by StreamAppendExpected when the stream's
	// current version does not match the expected version. This is the
	// optimistic concurrency sentinel for the StreamLogBackend.
	ErrVersionConflict = errors.New("metaengine: version conflict")
)
