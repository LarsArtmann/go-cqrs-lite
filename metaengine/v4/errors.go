package metaengine

import (
	"errors"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
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
	errNoQueryForName      = errors.New("no query declared for name")
	errUnsupportedPattern  = errors.New("unsupported read pattern")
	errUnknownFoldKind     = errors.New("unknown fold kind")

	// Construction-time errors.
	errUnsupportedDurability = errors.New("unsupported durability tier")
	errUnsupportedCellVersioning = errors.New(
		"unsupported cell versioning",
	)
	errExecuteTypeMismatch   = errors.New(
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

	errUnsupportedVectorFilters = errors.New(
		"engine does not support metadata-filtered vector searches (VectorFilterBackend)",
	)
	errUndirectedGraph = errors.New(
		"engine does not support undirected graph traversal (GraphNeighborsUndirected)",
	)
	errEdgeRemoval = errors.New(
		"engine does not support graph edge removal (GraphRemoveEdge)",
	)

	errUnsupportedSpatialReads = errors.New("engine does not support Spatial reads")
	errUnsupportedSpatialOps   = errors.New("engine does not support Spatial operations")

	// Verify / consistency errors live in the exported block below
	// (ErrNoEventLog, ErrNoQueryDecls, ErrCollectionCountMismatch,
	// ErrVerifyDrift): an exported method returning a sentinel callers cannot
	// match is a dead end, so these are classified AND exported.

	// SwapEngine error.
	errSwapEngineNotFound = errors.New("metaengine.SwapEngine: engine not found")

	// Scan exactness error.
	errScanTruncated = errors.New("metaengine: scan result was truncated")

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

	// ErrNoQueryForName is returned when no registered query matches the
	// query name passed to ExecuteQueryByName/ExecuteTypedByName.
	ErrNoQueryForName = errNoQueryForName

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

	// ErrUnsupportedDurability is returned at engine construction when the
	// requested DurabilityTier is invalid or not implemented by the driver.
	// See ValidateDurabilityTier and RejectDurabilityTier.
	ErrUnsupportedDurability = errUnsupportedDurability

	// ErrUnsupportedCellVersioning is returned at engine construction when
	// DriverConfig.CellVersioning names a retention policy but the driver
	// cannot version cells. See RejectCellVersioning.
	ErrUnsupportedCellVersioning = errUnsupportedCellVersioning

	// ErrScanTruncated is returned by TypedReader.Scan/ScanPage when
	// WithExact was requested and the engine reported more matching rows than
	// the scan limit returned. It makes an otherwise silent truncation loud:
	// callers that need the complete set can raise the limit or use WithLimit(0).
	ErrScanTruncated = errScanTruncated

	// --- Verify / consistency errors ---
	//
	// Store.Verify can end in four structurally different ways. They are
	// exported AND classified so a health/doctor command can branch on the
	// outcome with errors.Is instead of string matching, and tell "verification
	// cannot run here" (the first two) apart from the drift it exists to
	// detect (the last two). See also Store.EventLogAttached.

	// ErrNoEventLog is returned by Store.Verify and Store.Backfill when no
	// event log is attached (WithEventLog was never called — under
	// system.New there is none, so both are structural no-ops there).
	// Rejection family: retrying cannot help, the deployment has to change.
	ErrNoEventLog error = errorfamily.NewRejection(
		"metaengine.no_event_log",
		"metaengine: no event log attached — call WithEventLog first",
	)

	// ErrNoQueryDecls is returned by Store.Verify when the store holds no
	// query declarations to replay against (Plan stored none).
	ErrNoQueryDecls error = errorfamily.NewRejection(
		"metaengine.no_query_decls",
		"metaengine.Verify: no query declarations stored",
	)

	// ErrCollectionCountMismatch is returned by Store.Verify when the replayed
	// store materialized fewer collections than the live store.
	ErrCollectionCountMismatch error = errorfamily.NewCorruption(
		"metaengine.collection_count_mismatch",
		"metaengine.Verify: collection count mismatch",
	)

	// ErrVerifyDrift is returned (wrapped, with the live/replay row counts) by
	// Store.Verify when a collection's live row count disagrees with the
	// replay — the drift Verify exists to find. Corruption family: the
	// projection state no longer follows from its journal.
	ErrVerifyDrift error = errorfamily.NewCorruption(
		"metaengine.verify_drift",
		"metaengine.Verify: collection row-count drift",
	)
)
