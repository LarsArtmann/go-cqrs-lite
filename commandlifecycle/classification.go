package commandlifecycle

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// DefaultRejectionFamilies returns the error families classified as business
// rejections: [errorfamily.Rejection] (bad input, unauthorized, not found)
// and [errorfamily.Conflict] (version mismatch, duplicate, state-machine
// violation). Both mean "no state changed; the caller must act" — the audit
// answer "rejected by rule", as opposed to "broke" (transient, corruption,
// infrastructure, orchestration), which keeps flowing to command.failed and
// the dead-letter queue.
//
// The default pairs with retry middleware: errorfamily.IsRetryable retries
// only Transient, so rejected commands are never re-attempted.
func DefaultRejectionFamilies() []errorfamily.Family {
	return []errorfamily.Family{errorfamily.Rejection, errorfamily.Conflict}
}

// WithRejectionFamilies overrides which error families emit command.rejected
// instead of command.failed. Pass families not in [DefaultRejectionFamilies]
// to widen the contract (e.g. treat Corruption as a rejection in a batch tool
// whose handlers validate payload integrity up front). An empty list disables
// rejection classification entirely: every error is a failure.
func WithRejectionFamilies(families ...errorfamily.Family) RecorderOption {
	return func(r *Recorder) { r.rejectionFamilies = families }
}

// IsRejection reports whether err classifies into the recorder's rejection
// families. Nil errors are never rejections. Unclassified errors default to
// [errorfamily.Transient] (errorfamily's fail-open contract) and are therefore
// failures, not rejections.
func (r *Recorder) IsRejection(err error) bool {
	if err == nil {
		return false
	}

	family := errorfamily.Classify(err)
	for _, rejection := range r.rejectionFamilies {
		if family == rejection {
			return true
		}
	}

	return false
}
