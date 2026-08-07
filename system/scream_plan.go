package system

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// CheckPlanSafety compares the current projection plan against a pinned manifest.
// This is the core scream-store mechanism: if a projection query is removed or
// its ADT changes between restarts, the system refuses to start (SCREAM) because
// the existing data would be orphaned or incompatible.
//
// When the manifest file does not exist, this is treated as a first deployment
// (ADVISORY) and the current plan is saved as the new manifest.
//
// When checks pass (no SCREAM-tier violations), the current plan is saved to
// manifestPath for the next startup.
func CheckPlanSafety(
	_ context.Context,
	currentPlan *metaengine.SerializablePlan,
	manifestPath string,
) (*ScreamReport, error) {
	report := &ScreamReport{}

	if manifestPath == "" || currentPlan == nil {
		return report, nil
	}

	// Attempt to load the pinned manifest.
	prevManifest, err := metaengine.LoadManifest(manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// First deployment — no pinned plan to diff against.
			report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
				Tier:   TierAdvisory,
				Rule:   "plan:first-deployment",
				Detail: "no pinned manifest found — this is the first deployment; the current plan will be pinned on successful startup",
			})

			if saveErr := savePlanManifest(currentPlan, manifestPath); saveErr != nil {
				return nil, fmt.Errorf("system: save initial manifest: %w", saveErr)
			}

			return report, nil
		}

		return nil, fmt.Errorf("system: load manifest %q: %w", manifestPath, err)
	}

	// Diff the pinned plan against the current plan.
	diff := metaengine.PlanDiff(prevManifest.Plan, currentPlan)
	classifyPlanDiff(diff, report)

	// Verify the manifest's fingerprint integrity.
	if ok, fpErr := prevManifest.VerifyFingerprint(); fpErr != nil {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier:   TierWarnOverride,
			Rule:   "plan:manifest-corrupt",
			Detail: fmt.Sprintf("manifest fingerprint verification failed: %v", fpErr),
		})
	} else if !ok {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier:   TierScream,
			Rule:   "plan:manifest-tampered",
			Detail: "manifest file was modified after creation — fingerprint does not match the stored plan",
		})
	}

	// If no SCREAM-tier violations, save the current plan as the new manifest.
	if !report.HasErrors() {
		if saveErr := savePlanManifest(currentPlan, manifestPath); saveErr != nil {
			return nil, fmt.Errorf("system: save manifest: %w", saveErr)
		}
	}

	return report, nil
}

// classifyPlanDiff converts a PlanDiffResult into scream diagnostics.
func classifyPlanDiff(diff *metaengine.PlanDiffResult, report *ScreamReport) {
	for _, name := range diff.QueriesRemoved {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier: TierScream,
			Rule: "plan:query-removed:" + name,
			Detail: fmt.Sprintf(
				"projection query %q was removed — existing data is orphaned and cannot be replayed",
				name,
			),
		})
	}

	for _, change := range diff.QueriesChanged {
		if change.OldADT != change.NewADT {
			report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
				Tier: TierScream,
				Rule: "plan:adt-changed:" + change.Name,
				Detail: fmt.Sprintf(
					"projection query %q changed ADT from %q to %q — incompatible data model",
					change.Name,
					change.OldADT,
					change.NewADT,
				),
			})
		} else if change.OldEngine != change.NewEngine {
			report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
				Tier:   TierWarnOverride,
				Rule:   "plan:engine-changed:" + change.Name,
				Detail: fmt.Sprintf("projection query %q moved from engine %q to %q — data may need backfill", change.Name, change.OldEngine, change.NewEngine),
			})
		}
	}

	for _, name := range diff.QueriesAdded {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier:   TierAdvisory,
			Rule:   "plan:query-added:" + name,
			Detail: fmt.Sprintf("new projection query %q added — will backfill from the event log on startup", name),
		})
	}

	for _, table := range diff.LayoutsRemoved {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier:   TierWarnOverride,
			Rule:   "plan:layout-removed:" + table,
			Detail: fmt.Sprintf("layout/table %q was removed — existing data in this table may be orphaned", table),
		})
	}

	for _, table := range diff.LayoutsAdded {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier:   TierAdvisory,
			Rule:   "plan:layout-added:" + table,
			Detail: fmt.Sprintf("new layout/table %q added — will be created on startup", table),
		})
	}

	for _, eng := range diff.EnginesRemoved {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier: TierWarnOverride,
			Rule: "plan:engine-removed:" + eng,
			Detail: fmt.Sprintf(
				"engine %q was removed from the projection pool — queries previously assigned to it may have moved",
				eng,
			),
		})
	}

	for _, eng := range diff.EnginesAdded {
		report.Diagnostics = append(report.Diagnostics, ScreamDiagnostic{
			Tier:   TierAdvisory,
			Rule:   "plan:engine-added:" + eng,
			Detail: fmt.Sprintf("engine %q was added to the projection pool — planner may reassign queries", eng),
		})
	}
}

func savePlanManifest(plan *metaengine.SerializablePlan, path string) error {
	manifest, err := metaengine.NewManifest(plan)
	if err != nil {
		return fmt.Errorf("create manifest: %w", err)
	}

	return metaengine.SaveManifest(path, manifest)
}
