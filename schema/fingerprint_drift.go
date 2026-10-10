package schema

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// DriftHook observes one shape-drift finding: a stored event whose
// [FingerprintMetadataKey] stamp disagrees with the currently declared
// fingerprint for its type. It receives the event, the stamped (stored)
// fingerprint, and the declared one. Hooks run inline on the read path —
// keep them fast (slog/otel emit, counter bump), never blocking.
type DriftHook func(evt event.Event, stamped, declared string)

// FingerprintDrift returns an [event.SourceTransform] that compares every
// loaded event's fingerprint stamp against the declared set and invokes
// onDrift on mismatch. Compose it BESIDE the upcast chain — drift detection
// observes; the chain still normalizes (compose via
// event.ComposeSourceTransforms or a DecorateStore chain so reads upcast
// first, then report). Semantics:
//
//   - stamped ≠ declared → onDrift fires (the event still passes through —
//     advisory by design; the upcast chain is the correctness mechanism)
//   - NO stamp (pre-stamping events, undeclared writers) → silent accept
//     (absent = accept, the same rule as snapshot stamps)
//   - undeclared event type → silent accept (nothing to compare against)
//
// Hard mode (fail the read) is [FingerprintDriftHard]; the advisory ledger is
// the burn-in surface — flip to hard once the drift counter has been quiet
// for a full deployment cycle.
func FingerprintDrift(declared []EventSchema, onDrift DriftHook) event.SourceTransform {
	if onDrift == nil {
		return nil
	}

	fingerprints := declaredFingerprints(declared)

	return func(events []event.Event) ([]event.Event, error) {
		for _, evt := range events {
			stamped, isStamped := stampedFingerprint(evt)
			if !isStamped {
				continue
			}

			if declaredFp, isDeclared := fingerprints[evt.Type()]; isDeclared && stamped != declaredFp {
				onDrift(evt, stamped, declaredFp)
			}
		}

		return events, nil
	}
}

// FingerprintDriftHard is the hard-mode twin of [FingerprintDrift]: a
// stamp/declaration mismatch fails the read with a Corruption error instead
// of invoking a hook. Unstamped events and undeclared types still pass
// (absent = accept) — hard mode catches REGRESSIONS after burn-in, it does
// not punish history.
func FingerprintDriftHard(declared []EventSchema) event.SourceTransform {
	fingerprints := declaredFingerprints(declared)

	return func(events []event.Event) ([]event.Event, error) {
		for _, evt := range events {
			stamped, isStamped := stampedFingerprint(evt)
			if !isStamped {
				continue
			}

			declaredFp, isDeclared := fingerprints[evt.Type()]
			if isDeclared && stamped != declaredFp {
				return nil, errorfamily.NewCorruption(
					"schema.fingerprint_drift",
					"event "+evt.ID().String()+" ("+evt.Type().String()+") was written under shape "+
						stamped+" but the declared shape is "+declaredFp,
				)
			}
		}

		return events, nil
	}
}

func declaredFingerprints(declared []EventSchema) map[event.Type]string {
	fingerprints := make(map[event.Type]string, len(declared))
	for _, declaration := range declared {
		fingerprints[declaration.eventType] = declaration.Fingerprint()
	}

	return fingerprints
}

func stampedFingerprint(evt event.Event) (string, bool) {
	if evt.Metadata().Custom == nil {
		return "", false
	}

	stamped, ok := evt.Metadata().Custom[event.MetadataKey(FingerprintMetadataKey)]

	return stamped, ok && stamped != ""
}
