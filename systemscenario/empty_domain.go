package systemscenario

import (
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// logfSink is the slice of testing.TB the empty-domain check needs; a
// package-local interface keeps the check testable without a testing.TB stub.
type logfSink interface {
	Logf(format string, args ...any)
}

// warnOnEmptyDomain surfaces the false-confidence trap a 2026-10-24 fleet
// audit documented (CV): a test calls System with an EMPTY DomainConfig —
// usually because a wrapper owns the real wiring — and every Then* assertion
// passes against a hollow composition root. Wrapper-owned boots should
// construct the system in the test-fixture layer and hand it to Adopt
// instead, so the scenario exercises the production wiring.
//
// The warning is advisory (tb.Logf): a deliberately empty domain is
// legitimate for harness self-tests.
func warnOnEmptyDomain(tb logfSink, domain system.DomainConfig) {
	if !isEmptyDomain(domain) {
		return
	}

	tb.Logf(
		"systemscenario: WARNING: DomainConfig is empty (no commands, queries, " +
			"timers, projections, evolutions, events, or schema declarations) — " +
			"this scenario boots a HOLLOW composition root, so passing assertions " +
			"prove nothing about your domain. If a wrapper owns system.New, boot " +
			"it in the fixture and wrap it with systemscenario.Adopt(sys) instead.",
	)
}

// isEmptyDomain reports whether every domain-wiring slot of cfg is unset.
// Flags (DisableCoeffectValidation, StampSchemaFingerprints) and the
// ProjectionDecoder do not count: they tune behavior but declare nothing.
func isEmptyDomain(cfg system.DomainConfig) bool {
	return cfg.Commands == nil &&
		cfg.Queries == nil &&
		cfg.Timers == nil &&
		len(cfg.Projections) == 0 &&
		len(cfg.Evolutions) == 0 &&
		len(cfg.Events) == 0 &&
		len(cfg.Schema) == 0
}
