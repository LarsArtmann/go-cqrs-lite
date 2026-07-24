package benchkit

import errorfamily "github.com/larsartmann/go-error-family"

// Classified errors for the benchkit package.
// Following the 5-family taxonomy from go-error-family:
// Rejection (not retryable, client error), Conflict (concurrent modification),
// Transient (retryable), Infrastructure (non-retryable backend failure),
// Corruption (data integrity violation).

var (
	// ErrInvalidConfig is returned when Config validation fails
	// (e.g. Streams=0, EventsPerStream=0).
	ErrInvalidConfig = errorfamily.NewRejection(
		"benchkit.invalid_config",
		"benchmark configuration is invalid",
	)

	// ErrFactoryFailed is returned when the Factory function returns an error.
	ErrFactoryFailed = errorfamily.NewInfrastructure(
		"benchkit.factory_failed",
		"backend factory failed",
	)

	// ErrNilBundle is returned when the Factory returns a nil *stack.Bundle.
	ErrNilBundle = errorfamily.NewInfrastructure(
		"benchkit.nil_bundle",
		"factory returned nil bundle",
	)

	// ErrIncompleteBundle is returned when the Bundle is missing required
	// capabilities (EventSink or EventSource).
	ErrIncompleteBundle = errorfamily.NewInfrastructure(
		"benchkit.incomplete_bundle",
		"bundle is missing required event sink or source",
	)

	// ErrWarmupFailed is returned when the warmup phase encounters an error.
	ErrWarmupFailed = errorfamily.NewTransient(
		"benchkit.warmup_failed",
		"warmup phase failed",
	)
)
