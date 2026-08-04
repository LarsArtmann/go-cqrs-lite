package metaengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"slices"
	"time"
)

// ─── PlanDiff (T15) ───

// PlanDiffResult describes the differences between two SerializablePlans.
type PlanDiffResult struct {
	EnginesAdded   []string
	EnginesRemoved []string
	QueriesAdded   []string
	QueriesRemoved []string
	QueriesChanged []QueryChange
	LayoutsAdded   []string
	LayoutsRemoved []string
}

// QueryChange describes a query whose assignment changed between plans.
type QueryChange struct {
	Name      string
	OldEngine string
	NewEngine string
	OldADT    ADT
	NewADT    ADT
}

// IsEmpty returns true if the diff has no changes.
func (d *PlanDiffResult) IsEmpty() bool {
	return len(d.EnginesAdded) == 0 && len(d.EnginesRemoved) == 0 &&
		len(d.QueriesAdded) == 0 && len(d.QueriesRemoved) == 0 &&
		len(d.QueriesChanged) == 0 &&
		len(d.LayoutsAdded) == 0 && len(d.LayoutsRemoved) == 0
}

// PlanDiff compares two SerializablePlans and returns the differences.
func PlanDiff(prev, current *SerializablePlan) *PlanDiffResult {
	diff := &PlanDiffResult{}

	// Engine diffs.
	prevEngines := toSet(prev.Engines)
	currEngines := toSet(current.Engines)

	diff.EnginesAdded = setDiff(currEngines, prevEngines)
	diff.EnginesRemoved = setDiff(prevEngines, currEngines)

	// Query diffs.
	prevQueries := make(map[string]SerializableQuery, len(prev.Queries))
	for _, q := range prev.Queries {
		prevQueries[q.Name] = q
	}

	currQueries := make(map[string]SerializableQuery, len(current.Queries))
	for _, q := range current.Queries {
		currQueries[q.Name] = q
	}

	for name, cq := range currQueries {
		if pq, exists := prevQueries[name]; !exists {
			diff.QueriesAdded = append(diff.QueriesAdded, name)
		} else if pq.Engine != cq.Engine || pq.ADT != cq.ADT {
			diff.QueriesChanged = append(diff.QueriesChanged, QueryChange{
				Name:      name,
				OldEngine: pq.Engine,
				NewEngine: cq.Engine,
				OldADT:    pq.ADT,
				NewADT:    cq.ADT,
			})
		}
	}

	for name := range prevQueries {
		if _, exists := currQueries[name]; !exists {
			diff.QueriesRemoved = append(diff.QueriesRemoved, name)
		}
	}

	// Layout diffs.
	prevLayouts := make(map[string]bool, len(prev.Layouts))
	for _, l := range prev.Layouts {
		prevLayouts[l.Table] = true
	}

	currLayouts := make(map[string]bool, len(current.Layouts))
	for _, l := range current.Layouts {
		currLayouts[l.Table] = true
	}

	for table := range currLayouts {
		if !prevLayouts[table] {
			diff.LayoutsAdded = append(diff.LayoutsAdded, table)
		}
	}

	for table := range prevLayouts {
		if !currLayouts[table] {
			diff.LayoutsRemoved = append(diff.LayoutsRemoved, table)
		}
	}

	// Sort for deterministic output.
	slices.Sort(diff.EnginesAdded)
	slices.Sort(diff.EnginesRemoved)
	slices.Sort(diff.QueriesAdded)
	slices.Sort(diff.QueriesRemoved)
	slices.Sort(diff.LayoutsAdded)
	slices.Sort(diff.LayoutsRemoved)

	return diff
}

// ─── PlanFingerprint (T16) ───

// PlanFingerprint returns a canonical SHA-256 hash of a SerializablePlan.
// The hash is stable for structurally identical plans, enabling pinned-plan
// verification and drift detection.
func PlanFingerprint(plan *SerializablePlan) (string, error) {
	data, err := json.Marshal(plan)
	if err != nil {
		return "", fmt.Errorf("metaengine.PlanFingerprint: marshal: %w", err)
	}

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:]), nil
}

// MustPlanFingerprint is like PlanFingerprint but panics on error.
// Useful for tests where the plan is known to be valid.
func MustPlanFingerprint(plan *SerializablePlan) string {
	fp, err := PlanFingerprint(plan)
	if err != nil {
		panic(err)
	}

	return fp
}

// ─── Manifest (T17) ───

// Manifest bundles a SerializablePlan with metadata for persistence.
// The scream store uses manifests to detect plan drift across restarts.
type Manifest struct {
	Plan        *SerializablePlan `json:"plan"`
	Fingerprint string            `json:"fingerprint"`
	CreatedAt   time.Time         `json:"created_at"`
	Version     int               `json:"version"`
}

// NewManifest creates a Manifest from a SerializablePlan, computing the
// fingerprint automatically.
func NewManifest(plan *SerializablePlan) (*Manifest, error) {
	fp, err := PlanFingerprint(plan)
	if err != nil {
		return nil, err
	}

	return &Manifest{
		Plan:        plan,
		Fingerprint: fp,
		CreatedAt:   time.Now().UTC(),
		Version:     1,
	}, nil
}

// SaveManifest writes a Manifest to a file as JSON.
func SaveManifest(path string, m *Manifest) error {
	data, err := json.Marshal(m, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("metaengine.SaveManifest: marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("metaengine.SaveManifest: write %q: %w", path, err)
	}

	return nil
}

// LoadManifest reads a Manifest from a JSON file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("metaengine.LoadManifest: read %q: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("metaengine.LoadManifest: unmarshal: %w", err)
	}

	return &m, nil
}

// VerifyFingerprint checks whether the manifest's plan still matches its
// recorded fingerprint. Returns true if the plan is unchanged.
func (m *Manifest) VerifyFingerprint() (bool, error) {
	fp, err := PlanFingerprint(m.Plan)
	if err != nil {
		return false, err
	}

	return fp == m.Fingerprint, nil
}

// ─── helpers ───

func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}

	return set
}

func setDiff(a, b map[string]bool) []string {
	var diff []string

	for item := range a {
		if !b[item] {
			diff = append(diff, item)
		}
	}

	return diff
}
