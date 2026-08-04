package system

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ScreamTier classifies the severity of a safety violation.
type ScreamTier string

const (
	// TierScream blocks startup — the change would cause data loss or corruption.
	TierScream ScreamTier = "SCREAM"
	// TierWarnOverride logs loudly and requires explicit operator acknowledgment.
	TierWarnOverride ScreamTier = "WARN+OVERRIDE"
	// TierAdvisory is informational; dashboard shows yellow but startup proceeds.
	TierAdvisory ScreamTier = "ADVISORY"
)

// ScreamDiagnostic is one safety finding from the scream store.
type ScreamDiagnostic struct {
	Tier   ScreamTier
	Rule   string
	Detail string
}

// ScreamReport aggregates all safety diagnostics for a deployment.
type ScreamReport struct {
	Diagnostics []ScreamDiagnostic
}

// HasErrors returns true if any SCREAM-tier diagnostics exist.
func (r *ScreamReport) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Tier == TierScream {
			return true
		}
	}

	return false
}

// HasWarnings returns true if any WARN+OVERRIDE diagnostics exist (unACKed).
func (r *ScreamReport) HasWarnings() bool {
	for _, d := range r.Diagnostics {
		if d.Tier == TierWarnOverride {
			return true
		}
	}

	return false
}

// ErrUnsafeChange is returned when the scream store detects a SCREAM-tier
// violation. The system refuses to start.
var ErrUnsafeChange = errors.New("system: unsafe deployment change detected (SCREAM-tier)")

// CheckSafety validates the deployment config against safety rules.
// This is the initial implementation — the full scream store will diff
// against a pinned SerializablePlan manifest.
func CheckSafety(_ context.Context, deployment DeploymentConfig) (*ScreamReport, error) {
	report := &ScreamReport{}

	// Rule: source-of-truth must use a persistent engine (not "memory")
	for _, inst := range deployment.Instances {
		if isSourceOfTruth(inst.Role) {
			engineName := inst.Engine
			if engineName == "" && len(inst.Engines) > 0 {
				engineName = inst.Engines[0]
			}

			if engCfg, ok := deployment.Engines[engineName]; ok {
				if engCfg.Driver == "memory" && inst.Durability != DurabilityRelaxed {
					report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
						Tier: TierWarnOverride,
						Rule: "volatile-source-of-truth",
						Detail: fmt.Sprintf(
							"instance %q uses volatile 'memory' driver for source-of-truth — data will be lost on restart",
							inst.Role,
						),
					})
				}
			}
		}
	}

	// Rule: durability downgrade
	for _, inst := range deployment.Instances {
		if inst.Durability == DurabilityRelaxed && isSourceOfTruth(inst.Role) {
			ruleKey := fmt.Sprintf("durability-downgrade:%s", inst.Role)
			if !isAcknowledged(deployment, ruleKey) {
				report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
					Tier: TierWarnOverride,
					Rule: ruleKey,
					Detail: fmt.Sprintf(
						"instance %q uses Relaxed durability — data loss possible on crash",
						inst.Role,
					),
				})
			}
		}
	}

	return report, nil
}

func isAcknowledged(cfg DeploymentConfig, ruleKey string) bool {
	for _, ack := range cfg.AcknowledgeWarnings {
		if strings.EqualFold(ack, ruleKey) {
			return true
		}
	}

	return false
}

// ScreamReportAccess returns the safety report for the running system.
func (s *System) ScreamReport() *ScreamReport {
	report, _ := CheckSafety(context.Background(), s.deployment)

	return report
}
