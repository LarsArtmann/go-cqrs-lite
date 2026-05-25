package testhelpers

import (
	"strings"
	"sync"
	"testing"
	"time"
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

// AssertLen asserts that the slice has the expected length.
func AssertLen[T any](t *testing.T, name string, slice []T, want int) {
	t.Helper()

	if len(slice) != want {
		t.Errorf("%s = %d, want %d", name, len(slice), want)
	}
}

// AssertLenFatal asserts that the slice has the expected length, fataling on mismatch.
func AssertLenFatal[T any](t *testing.T, name string, slice []T, want int) {
	t.Helper()

	if len(slice) != want {
		t.Fatalf("%s = %d, want %d", name, len(slice), want)
	}
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

// AssertNotContains fails if s contains substr.
func AssertNotContains(t *testing.T, s, substr, msg string) {
	t.Helper()

	if strings.Contains(s, substr) {
		t.Error(msg)
	}
}
