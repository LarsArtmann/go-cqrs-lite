package metaengine_test

import (
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestFilterBindValue pins the issue #62 coercion contract: time.Time filter
// binds render exactly like encoding/json's marshaling of the same instant
// (RFC3339Nano, offset-preserving), FilterIn lists coerce element-wise, and
// every other value passes through untouched.
func TestFilterBindValue(t *testing.T) {
	t.Parallel()

	utc := time.Date(2026, 10, 11, 14, 30, 5, 0, time.UTC)
	if got := metaengine.FilterBindValue(utc); got != "2026-10-11T14:30:05Z" {
		t.Errorf("UTC time bind = %v, want 2026-10-11T14:30:05Z", got)
	}

	zone := time.FixedZone("+02:00", 2*60*60)
	located := time.Date(2026, 10, 11, 16, 30, 5, 0, zone)
	if got := metaengine.FilterBindValue(located); got != "2026-10-11T16:30:05+02:00" {
		t.Errorf("offset time bind = %v, want 2026-10-11T16:30:05+02:00", got)
	}

	nano := time.Date(2026, 10, 11, 14, 30, 5, 123456789, time.UTC)
	if got := metaengine.FilterBindValue(nano); got != "2026-10-11T14:30:05.123456789Z" {
		t.Errorf("nanosecond time bind = %v, want 2026-10-11T14:30:05.123456789Z", got)
	}

	if json, err := utc.MarshalJSON(); err != nil || string(json) != `"2026-10-11T14:30:05Z"` {
		t.Fatalf("json.Marshal sanity: %s (%v)", json, err)
	}

	list := []any{"open", utc, 3}
	bound := metaengine.FilterBindValue(list)
	boundList, ok := bound.([]any)
	if !ok || len(boundList) != 3 {
		t.Fatalf("list bind type = %T (%v), want []any len 3", bound, bound)
	}
	if boundList[0] != "open" || boundList[1] != "2026-10-11T14:30:05Z" || boundList[2] != 3 {
		t.Errorf("list bind = %v, want [open 2026-10-11T14:30:05Z 3]", boundList)
	}

	plain := []any{"open", "closed"}
	if got := metaengine.FilterBindValue(plain); len(got.([]any)) != 2 {
		t.Errorf("time-free list must pass through, got %v", got)
	}

	if got := metaengine.FilterBindValue("open"); got != "open" {
		t.Errorf("string passthrough = %v", got)
	}
	if got := metaengine.FilterBindValue(42); got != 42 {
		t.Errorf("int passthrough = %v", got)
	}
	if got := metaengine.FilterBindValue(nil); got != nil {
		t.Errorf("nil passthrough = %v", got)
	}
}
