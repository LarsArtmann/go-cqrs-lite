package main

import (
	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// adoptScorecardWaivers folds waiver-adjudicated adoption findings into the
// suppressed set: a scorecard waiver in .cqrs-lint.json is the recorded
// refusal, so the finding renders under --show-suppressed with the waiver's
// reason + trigger instead of forcing the baseline or a package-clause
// directive to hold it. A waiver load failure leaves findings active.
func adoptScorecardWaivers(
	findings []finding.Finding,
	cfg *AppConfig,
	suppressed []finding.Finding,
) (active, allSuppressed []finding.Finding) {
	waivers, err := resolveScorecardWaivers(cfg)
	if err != nil {
		return findings, suppressed
	}

	active, waived := filterWaivedAdoption(findings, waivers)

	return active, append(suppressed, waived...)
}

// waiverAdoptionRules maps scorecard waiver keys to the adoption (F-series)
// rule that fires for the same module. Both surfaces adjudicate the same
// question — "why doesn't this project use module X?" — so a recorded waiver
// must also silence that module's adoption finding; otherwise the baseline
// becomes the silencer waivers explicitly refuse to be. Non-adoption findings
// for a waived module (none today) would keep firing.
var waiverAdoptionRules = map[string]string{ //nolint:gochecknoglobals // static lookup table
	"otel":        "F003",
	"prometheus":  "F004",
	"schema":      "F005",
	"encryption":  "F006",
	"idempotency": "F007",
	"scheduling":  "F009",
	"listing":     "F016",
}

// filterWaivedAdoption moves adoption findings whose module carries a
// recorded scorecard waiver into the suppressed set, annotated with the
// waiver's reason and revisit trigger. The refusal stays auditable: the
// findings appear under --show-suppressed carrying the waiver, and the
// scorecard still renders its WAIVED row. A waiver load failure leaves
// findings active (loud, not silently suppressed).
func filterWaivedAdoption(
	findings []finding.Finding,
	waivers []analyzer.ScorecardWaiver,
) (active, waived []finding.Finding) {
	ruleWaivers := make(map[string]analyzer.ScorecardWaiver, len(waivers))

	for _, w := range waivers {
		if rule, ok := waiverAdoptionRules[w.Key]; ok {
			ruleWaivers[rule] = w
		}
	}

	if len(ruleWaivers) == 0 {
		return findings, nil
	}

	active = make([]finding.Finding, 0, len(findings))
	waived = make([]finding.Finding, 0)

	for _, f := range findings {
		w, ok := ruleWaivers[string(f.Rule)]
		if !ok {
			active = append(active, f)

			continue
		}

		reason := "scorecard waiver: " + w.Reason
		if w.Trigger != "" {
			reason += " — revisit when: " + w.Trigger
		}

		f.Suppression = &finding.Suppression{
			Kind:   finding.SuppressionInSource,
			Rule:   f.Rule,
			Reason: reason,
		}
		waived = append(waived, f)
	}

	return active, waived
}
