package bigtableengine

import (
	"testing"
	"time"

	"cloud.google.com/go/bigtable"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestGCPolicyFor pins the RetentionPolicy → column-family GC policy
// translation (issue #131, ADR-0141 §1): BigTable versions cells natively,
// so the operator's retention knob maps onto the GC policy — versions-only
// → MaxVersionsPolicy, age-only → MaxAgePolicy, both → IntersectionPolicy
// (a version survives only when it satisfies both), zero → nil (no GC
// policy, every version kept).
func TestGCPolicyFor(t *testing.T) {
	if got := gcPolicyFor(metaengine.RetentionPolicy{}); got != nil {
		t.Fatalf("zero policy = %T, want nil (no GC policy)", got)
	}

	if _, ok := gcPolicyFor(metaengine.RetentionPolicy{MaxVersions: 4}).(bigtable.MaxVersionsGCPolicy); !ok {
		t.Fatalf(
			"versions-only policy type = %T, want MaxVersionsGCPolicy",
			gcPolicyFor(metaengine.RetentionPolicy{MaxVersions: 4}),
		)
	}

	if _, ok := gcPolicyFor(metaengine.RetentionPolicy{MaxAge: time.Hour}).(bigtable.MaxAgeGCPolicy); !ok {
		t.Fatalf(
			"age-only policy type = %T, want MaxAgeGCPolicy",
			gcPolicyFor(metaengine.RetentionPolicy{MaxAge: time.Hour}),
		)
	}

	both := metaengine.RetentionPolicy{MaxVersions: 4, MaxAge: time.Hour}
	if _, ok := gcPolicyFor(both).(bigtable.IntersectionGCPolicy); !ok {
		t.Fatalf("both-knobs policy type = %T, want IntersectionGCPolicy", gcPolicyFor(both))
	}
}
