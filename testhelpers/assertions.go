package testhelpers

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// FakeMetrics is a metrics collector for testing.
type FakeMetrics struct {
	mu        sync.Mutex
	Records   []string
	Durations []time.Duration
}

// Observe records a metric observation.
func (m *FakeMetrics) Observe(name string, duration time.Duration, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Records = append(m.Records, name)
	m.Durations = append(m.Durations, duration)
}

// AssertCallOrder asserts the call order matches expected.
func AssertCallOrder(t *testing.T, callOrder, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)

			break
		}
	}
}

// AssertMetricRecord asserts the metrics recorder has exactly one record with the given name.
func AssertMetricRecord(t *testing.T, m *FakeMetrics, wantName string) {
	t.Helper()

	if len(m.Records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(m.Records))
	}

	if m.Records[0] != wantName {
		t.Errorf("expected %s, got %s", wantName, m.Records[0])
	}
}

// assertSliceLen is a shared helper for AssertLen and AssertLenFatal.
func assertSliceLen[T any](t *testing.T, name string, slice []T, want int, fail func(string, ...any)) {
	t.Helper()

	if len(slice) != want {
		fail("%s = %d, want %d", name, len(slice), want)
	}
}

// AssertLen asserts that the slice has the expected length.
func AssertLen[T any](t *testing.T, name string, slice []T, want int) {
	assertSliceLen(t, name, slice, want, t.Errorf)
}

// AssertLenFatal asserts that the slice has the expected length, fataling on mismatch.
func AssertLenFatal[T any](t *testing.T, name string, slice []T, want int) {
	assertSliceLen(t, name, slice, want, t.Fatalf)
}

// AssertNoError fails if err is not nil.
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError fails if err is nil.
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()

	if err == nil {
		t.Error(msg)
	}
}

// AssertErrorWithResult asserts result is nil, err is not nil, and err.Error() contains substr.
func AssertErrorWithResult(t *testing.T, result any, err error, substr string) {
	t.Helper()

	if result != nil && !reflect.ValueOf(result).IsNil() {
		t.Errorf("result = %v, want nil", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error = %q, want containing %q", err.Error(), substr)
	}
}

// AssertEqual fails if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()

	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertContains fails if s does not contain substr.
func AssertContains(t *testing.T, s, substr, msg string) {
	t.Helper()

	if !strings.Contains(s, substr) {
		t.Error(msg)
	}
}

// AssertPanics asserts that fn panics.
func AssertPanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	fn()
}

// AssertNotContains fails if s contains substr.
func AssertNotContains(t *testing.T, s, substr, msg string) {
	t.Helper()

	if strings.Contains(s, substr) {
		t.Error(msg)
	}
}

// AssertEventType asserts that events[index].Type() equals want.
func AssertEventType(t *testing.T, events []event.Event, index int, want string) {
	t.Helper()

	if got := string(events[index].Type()); got != want {
		t.Errorf("events[%d].Type = %q, want %s", index, got, want)
	}
}

// AssertEventVersion asserts that events[index].Version() equals want.
func AssertEventVersion(t *testing.T, events []event.Event, index, want int) {
	t.Helper()

	if got := int(events[index].Version()); got != want {
		t.Errorf("events[%d].Version = %d, want %d", index, got, want)
	}
}
