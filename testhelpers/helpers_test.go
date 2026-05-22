package testhelpers

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestEventHelpers_NewTestEvent(t *testing.T) {
	t.Parallel()

	evt, err := NewTestEvent()
	if err != nil {
		t.Fatalf("NewTestEvent: %v", err)
	}

	if evt.Type() != "test.evt" {
		t.Errorf("Type = %q, want test.evt", evt.Type())
	}
}

func TestEventHelpers_NewEvent(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := NewEvent(t, "user.created", aggID, "User", 1, []byte(`{}`))

	if evt.AggregateID() != aggID {
		t.Error("AggregateID mismatch")
	}
}

func TestEventHelpers_MakeEvent(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt, err := MakeEvent("user.created", aggID, "User", 1, nil)
	if err != nil {
		t.Fatalf("MakeEvent: %v", err)
	}

	if evt.AggregateID() != aggID {
		t.Error("AggregateID mismatch")
	}
}

func TestEventHelpers_QuickEvent(t *testing.T) {
	t.Parallel()

	aggID := id.NewAggregateID()
	evt := QuickEvent("user.created", aggID, "User", 1, nil)

	if evt.Type() != "user.created" {
		t.Errorf("Type = %q, want user.created", evt.Type())
	}
}

func TestTestMetrics_Observe(t *testing.T) {
	t.Parallel()

	m := &TestMetrics{}
	m.Observe("command.create", 100*time.Millisecond)

	AssertMetricRecord(t, m, "command.create")
}

func TestAssertCallOrder(t *testing.T) {
	t.Parallel()

	AssertCallOrder(t, []string{"a", "b", "c"}, []string{"a", "b", "c"})
}

func TestAssertLen(t *testing.T) {
	t.Parallel()

	AssertLen(t, "items", []string{"a", "b"}, 2)
}

func TestAssertLenFatal(t *testing.T) {
	t.Parallel()

	AssertLenFatal(t, "items", []int{1}, 1)
}

func TestAssertNoError(t *testing.T) {
	t.Parallel()

	AssertNoError(t, nil, "should be nil")
}

func TestAssertError(t *testing.T) {
	t.Parallel()

	AssertError(t, errors.New("fail"), "should have error")
}

func TestAssertEqual(t *testing.T) {
	t.Parallel()

	AssertEqual(t, 42, 42, "number")
	AssertEqual(t, "hello", "hello", "string")
}

func TestAssertContains(t *testing.T) {
	t.Parallel()

	AssertContains(t, "hello world", "world", "should contain")
}

func TestAssertNotContains(t *testing.T) {
	t.Parallel()

	AssertNotContains(t, "hello world", "xyz", "should not contain")
}
