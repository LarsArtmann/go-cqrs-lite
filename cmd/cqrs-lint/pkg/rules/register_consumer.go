package rules

import (
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/resilience"
	"github.com/larsartmann/go-finding"
)

// appendConsumerCoaching registers the consumer-coaching detectors: rules
// that coach CONSUMER code toward adopting library features. They are
// skipped when linting the library itself — the library cannot coach itself
// to "adopt" its own features, and consumer-architecture patterns (preset
// bypass, read-your-writes, signing defaults) are meaningless in library
// source.
func appendConsumerCoaching(
	detectors []finding.Detector,
	ctx *analyzer.AnalysisContext,
) []finding.Detector {
	return append(
		detectors,
		// Architecture (consumer-coaching)
		architecture.NewE008Detector(ctx),
		architecture.NewE009Detector(ctx),
		architecture.NewE010Detector(ctx),
		architecture.NewE011Detector(ctx),
		architecture.NewE012Detector(ctx),
		architecture.NewE013Detector(ctx),
		architecture.NewE014Detector(ctx),
		architecture.NewE015Detector(ctx),
		// Adoption (Feature Adoption Coaching)
		adoption.NewF001Detector(ctx),
		adoption.NewF002Detector(ctx),
		adoption.NewF003Detector(ctx),
		adoption.NewF004Detector(ctx),
		adoption.NewF005Detector(ctx),
		adoption.NewF006Detector(ctx),
		adoption.NewF007Detector(ctx),
		adoption.NewF008Detector(ctx),
		adoption.NewF009Detector(ctx),
		adoption.NewF010Detector(ctx),
		adoption.NewF011Detector(ctx),
		adoption.NewF012Detector(ctx),
		adoption.NewF013Detector(ctx),
		adoption.NewF014Detector(ctx),
		adoption.NewF015Detector(ctx),
		adoption.NewF016Detector(ctx),
		adoption.NewF017Detector(ctx),
		adoption.NewF018Detector(ctx),
		adoption.NewF019Detector(ctx),
		adoption.NewF020Detector(ctx),
		adoption.NewF021Detector(ctx),
		adoption.NewF022Detector(ctx),
		adoption.NewF023Detector(ctx),
		adoption.NewF024Detector(ctx),
		adoption.NewF025Detector(ctx),
		adoption.NewF026Detector(ctx),
		// Resilience (consumer-coaching)
		resilience.NewB029Detector(ctx),
		resilience.NewB030Detector(ctx),
		resilience.NewB031Detector(ctx),
		// Observability adoption
		adoption.NewF027Detector(ctx),
		adoption.NewF028Detector(ctx),
		adoption.NewF029Detector(ctx),
		// Deprecated-module migration coaching (ADR-0127)
		adoption.NewF030Detector(ctx),
		// Scan-truncation nudge (G-T14)
		adoption.NewF031Detector(ctx),
	)
}
