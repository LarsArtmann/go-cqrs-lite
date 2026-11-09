package analyzer

import (
	"errors"
	"fmt"
	"strings"
)

// ScorecardSettings is the scorecard subset of .cqrs-lint.json: recorded
// refusals for catalog modules that genuinely have no use case in this
// project, plus explicit usage pins for modules whose usage import detection
// cannot see. A waiver moves the row out of MISSING into a visible WAIVED
// partition — the refusal is recorded with its justification and revisit
// trigger instead of evaporating into prose the tool never sees again.
type ScorecardSettings struct {
	// Waivers records per-module adoption refusals. Validated by
	// ValidateScorecardWaivers at config load time; the scorecard
	// additionally refuses to waive rows that are used or irrelevant.
	Waivers []ScorecardWaiver `json:"waivers,omitempty"`

	// Modules pins module usage explicitly, overriding import detection for
	// composition-root consumers: `"modules": {"projectionhost": true}``
	// marks the module used when the system composition root wires it behind
	// an accessor (sys.ProjectionHost()) no import scan can attribute. Only
	// true is meaningful — false is rejected because recording a refusal is
	// what waivers are for, with a reason and a revisit trigger.
	Modules map[string]bool `json:"modules,omitempty"`
}

// ScorecardWaiver is one recorded refusal of one scorecard module.
//
// The scorecard exists to drive consumers toward maximal, most-modern
// go-cqrs-lite usage — a waiver is a deliberate, reviewable exception to
// that pressure, not a way to silence it: waived rows render as WAIVED with
// the reason and trigger attached, and the waiver file is committed with the
// project so every reviewer sees it.
type ScorecardWaiver struct {
	// Key is the catalog module key being waived (e.g. "graph"). Must name a
	// known scored catalog entry.
	Key string `json:"key"`
	// Reason is the mandatory justification. An unconditional refusal
	// without a reason is exactly the "do NOT chase the grade" moat the
	// scorecard exists to prevent.
	Reason string `json:"reason"`
	// Trigger names the condition under which the waiver must be
	// re-litigated (e.g. "daemon lands or a second writer appears").
	// Rendered with the waived row. Optional but strongly encouraged: a
	// waiver without a trigger is a permanent pin by accident.
	Trigger string `json:"trigger,omitempty"`
}

// ErrScorecardWaiver prefixes every waiver validation failure; dynamic
// detail (duplicate key, unknown module, ...) wraps this sentinel so
// callers can match with errors.Is.
var ErrScorecardWaiver = errors.New("scorecard waiver")

// scoredModuleKeys builds the scored-module lookup set shared by waiver and
// pin validation.
func scoredModuleKeys() map[ModuleKey]bool {
	scored := make(map[ModuleKey]bool, len(DefaultCatalog.Scored()))
	for _, e := range DefaultCatalog.Scored() {
		scored[e.Key] = true
	}
	return scored
}

// ValidateScorecardWaivers checks waivers against the default catalog:
// keys must be known scored modules, reasons must be non-empty, and keys
// must be unique. Returns the first violation wrapped with the key.
func ValidateScorecardWaivers(waivers []ScorecardWaiver) error {
	if len(waivers) == 0 {
		return nil
	}

	scored := scoredModuleKeys()

	seen := make(map[string]bool, len(waivers))
	for _, w := range waivers {
		switch {
		case w.Key == "":
			return fmt.Errorf("%w: key must not be empty", ErrScorecardWaiver)
		case seen[w.Key]:
			return fmt.Errorf("%w: duplicate key %q", ErrScorecardWaiver, w.Key)
		case !scored[ModuleKey(w.Key)]:
			return fmt.Errorf(
				"%w: unknown module key %q (must be a scored catalog key)",
				ErrScorecardWaiver,
				w.Key,
			)
		case strings.TrimSpace(w.Reason) == "":
			return fmt.Errorf("%w for %q: reason must not be empty", ErrScorecardWaiver, w.Key)
		}
		seen[w.Key] = true
	}

	return nil
}

// ValidateScorecardModulePins checks usage pins against the default catalog:
// keys must be known scored modules, and only true is accepted — a false pin
// is a refusal, which waivers record properly (with a reason and a revisit
// trigger) instead of a bare boolean.
func ValidateScorecardModulePins(pins map[string]bool) error {
	if len(pins) == 0 {
		return nil
	}

	scored := scoredModuleKeys()

	for key, used := range pins {
		switch {
		case !scored[ModuleKey(key)]:
			return fmt.Errorf(
				"%w: unknown module key %q (must be a scored catalog key)",
				ErrScorecardWaiver,
				key,
			)
		case !used:
			return fmt.Errorf(
				"%w for %q: false pins are not supported — record a waiver with a reason instead",
				ErrScorecardWaiver,
				key,
			)
		}
	}

	return nil
}
