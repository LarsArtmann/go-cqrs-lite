package metaengine

import "testing"

// TestPlanFingerprint_StructurallyIdenticalPlansAgree pins the fingerprint
// contract: "stable for structurally identical plans, enabling pinned-plan
// verification and drift detection". Two separately-built equal
// SerializablePlans must hash identically — the fingerprint feeds drift
// detection, so an iteration-order-dependent marshal would false-drift
// pinned plans across runs. (The Serializable* shapes carry no map fields
// today, so this guards the contract the moment one is added: with
// json.Deterministic(true) removed, a map-bearing plan flakes here within
// runs.)
func TestPlanFingerprint_StructurallyIdenticalPlansAgree(t *testing.T) {
	t.Parallel()

	build := func() *SerializablePlan {
		return &SerializablePlan{
			Engines: []string{"memory", "sqlite"},
			Queries: []SerializableQuery{
				{
					Name:        "users_by_email",
					ADT:         ADTMap,
					Engine:      "sqlite",
					Complexity:  string(ComplexityO1),
					LatencyMs:   0.4,
					Persistence: PersistencePersistent,
				},
				{
					Name:       "events_by_time",
					ADT:        ADTStreamLog,
					Engine:     "memory",
					Complexity: string(ComplexityOLogN),
					LatencyMs:  1.2,
				},
			},
			Layouts: []SerializableLayout{{Table: "users_planned", Columns: []string{"email", "status"}}},
		}
	}

	first := MustPlanFingerprint(build())

	for i := range 20 {
		if again := MustPlanFingerprint(build()); again != first {
			t.Fatalf("fingerprint of a structurally identical plan changed: run 0 = %s, run %d = %s",
				first, i, again)
		}
	}
}
