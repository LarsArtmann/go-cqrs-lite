package irohengine

import "errors"

// Sentinel errors for backend capabilities not implemented by the local engine.
// These are returned by passthrough methods when the wrapped engine does not
// implement the required backend interface.
var (
	ErrGraphBackendNotImplemented   = errors.New("local engine does not implement GraphBackend")
	ErrMapUpdaterNotImplemented     = errors.New("local engine does not implement MapUpdater")
	ErrScanBackendNotImplemented    = errors.New("local engine does not implement ScanBackend")
	ErrSearchBackendNotImplemented  = errors.New("local engine does not implement SearchBackend")
	ErrSpatialBackendNotImplemented = errors.New("local engine does not implement SpatialBackend")
	ErrVectorBackendNotImplemented  = errors.New("local engine does not implement VectorBackend")
)
