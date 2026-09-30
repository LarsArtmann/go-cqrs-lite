package testutil_test

import (
	"regexp"
	"testing"

	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

var eventPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]{2,63}$`)

func TestEventType_GeneratesValidStrings(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		val := testutil.EventType().Draw(t, "eventType")
		if !eventPattern.MatchString(val) {
			t.Errorf("EventType() = %q does not match pattern", val)
		}
	})
}

func TestAggregateType_GeneratesValidStrings(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		val := testutil.StreamType().Draw(t, "streamType")
		if !eventPattern.MatchString(val) {
			t.Errorf("StreamType() = %q does not match pattern", val)
		}
	})
}

func TestVersion_GeneratesInRange(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		val := testutil.Version().Draw(t, "version")
		if val < 1 {
			t.Errorf("Version() = %d, want >= 1", val)
		}
		if val > 10000 {
			t.Errorf("Version() = %d, want <= 10000", val)
		}
	})
}

func TestNonEmptyString_GeneratesNonEmpty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		val := testutil.NonEmptyString().Draw(t, "str")
		if len(val) == 0 {
			t.Error("NonEmptyString() returned empty string")
		}
	})
}
