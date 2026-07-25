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
	errADTNotSupported  = errors.New("query requires an ADT that no engine supports")
	errCannotInferADT   = errors.New("cannot infer ADT: no active folds (only skips)")
	errAmbiguousKey     = errors.New("ambiguous key: multiple fields of matching type")
	errNoKeyField       = errors.New("no field of matching type in event")
	errInvalidEventType = errors.New("metaengine.On: handler first param must match event type")

	// Dispatch-time errors.
	errNoQueryForInputType = errors.New("no query declared for input type")
	errUnsupportedPattern  = errors.New("unsupported read pattern")
	errUnknownFoldKind     = errors.New("unknown fold kind")
	errExecuteTypeMismatch = errors.New(
		"metaengine.ExecuteTyped: result type does not match expected",
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
)

// unsupportedEngine wraps a capability sentinel with the offending engine name.
func unsupportedEngine(base error, engineName string) error {
	return fmt.Errorf("%w: engine %s", base, engineName)
}
