package irohengine

import "errors"

// Sentinel errors for backend capabilities not implemented by the local engine.
// These are returned by passthrough methods when the wrapped engine does not
// implement the required backend interface.
var (
	ErrGraphBackendNotImplemented   = errors.New("local engine does not implement graph dispatch")
	ErrMapUpdaterNotImplemented     = errors.New("local engine does not implement MapUpdater")
	ErrScanBackendNotImplemented    = errors.New("local engine does not implement ScanBackend")
	ErrSearchBackendNotImplemented  = errors.New("local engine does not implement SearchBackend")
	ErrSpatialBackendNotImplemented = errors.New("local engine does not implement SpatialBackend")
	ErrVectorBackendNotImplemented  = errors.New("local engine does not implement VectorBackend")
)

// ErrTransportClosed reports a transport that can no longer deliver ops.
// Returned by LivenessReporter implementations (and surfaced through
// HealthCheck) after Close or network shutdown.
var ErrTransportClosed = errors.New("transport closed")
