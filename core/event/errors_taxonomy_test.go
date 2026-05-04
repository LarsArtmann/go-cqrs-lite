package event_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func TestFamily_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		family event.Family
		want   string
	}{
		{event.Rejection, "rejection"},
		{event.Conflict, "conflict"},
		{event.Transient, "transient"},
		{event.Corruption, "corruption"},
		{event.Infrastructure, "infrastructure"},
		{event.Family(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.family.String(); got != tc.want {
			t.Errorf("Family(%d).String() = %q, want %q", tc.family, got, tc.want)
		}
	}
}

func TestError_Constructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		err    *event.Error
		family event.Family
		code   string
	}{
		{
			"rejection",
			event.NewRejection("test.reject", "rejected"),
			event.Rejection,
			"test.reject",
		},
		{
			"conflict",
			event.NewConflict("test.conflict", "conflicted"),
			event.Conflict,
			"test.conflict",
		},
		{
			"transient",
			event.NewTransient("test.transient", "transient fail"),
			event.Transient,
			"test.transient",
		},
		{
			"corruption",
			event.NewCorruption("test.corrupt", "corrupted"),
			event.Corruption,
			"test.corrupt",
		},
		{
			"infrastructure",
			event.NewInfrastructure("test.infra", "down"),
			event.Infrastructure,
			"test.infra",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.err.Family != tc.family {
				t.Errorf("Family = %v, want %v", tc.err.Family, tc.family)
			}

			if tc.err.Code != tc.code {
				t.Errorf("Code = %q, want %q", tc.err.Code, tc.code)
			}

			if tc.err.Error() != tc.err.Message {
				t.Errorf("Error() = %q, want %q", tc.err.Error(), tc.err.Message)
			}

			if tc.err.Unwrap() != nil {
				t.Error("Unwrap() should be nil without WithCause")
			}
		})
	}
}

func TestError_WithCause(t *testing.T) {
	t.Parallel()

	inner := errors.New("inner error")

	err := event.NewTransient("test.retry", "retry me").WithCause(inner)
	if !errors.Is(err.Unwrap(), inner) {
		t.Error("WithCause did not set cause")
	}
}

func TestError_WithCause_chaining(t *testing.T) {
	t.Parallel()

	inner := errors.New("root")
	err := event.NewConflict("test.c", "msg").WithCause(inner)
	wrapped := fmt.Errorf("outer: %w", err)

	var extracted *event.Error
	if !errors.As(wrapped, &extracted) {
		t.Fatal("errors.As should find *event.Error in wrapped chain")
	}

	if extracted.Code != "test.c" {
		t.Errorf("Code = %q, want %q", extracted.Code, "test.c")
	}

	if !errors.Is(extracted.Unwrap(), inner) {
		t.Error("cause not preserved through chain")
	}
}

func TestClassify_sentinelMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want event.Family
	}{
		{"version conflict", event.ErrVersionConflict, event.Conflict},
		{"aggregate not found", event.ErrAggregateNotFound, event.Rejection},
		{"snapshot not found", event.ErrSnapshotNotFound, event.Rejection},
		{"store closed", event.ErrStoreClosed, event.Infrastructure},
		{"bus closed", event.ErrBusClosed, event.Infrastructure},
		{"snapshot store closed", event.ErrSnapshotStoreClosed, event.Infrastructure},
		{"nil projection", event.ErrNilProjection, event.Infrastructure},
		{"nil checkpoint", event.ErrNilCheckpointStore, event.Infrastructure},
		{"nil outbox", event.ErrNilOutbox, event.Infrastructure},
		{"nil bus", event.ErrNilBus, event.Infrastructure},
		{"already started", event.ErrAlreadyStarted, event.Infrastructure},
		{"duplicate projection", event.ErrDuplicateProjection, event.Conflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := event.Classify(tc.err); got != tc.want {
				t.Errorf("Classify(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestClassify_typedError(t *testing.T) {
	t.Parallel()

	err := event.NewRejection("test.r", "bad input")
	if got := event.Classify(err); got != event.Rejection {
		t.Errorf("Classify(typed error) = %v, want %v", got, event.Rejection)
	}
}

func TestClassify_wrappedTypedError(t *testing.T) {
	t.Parallel()

	inner := event.NewConflict("test.c", "conflict")

	wrapped := fmt.Errorf("handler failed: %w", inner)
	if got := event.Classify(wrapped); got != event.Conflict {
		t.Errorf("Classify(wrapped typed error) = %v, want %v", got, event.Conflict)
	}
}

func TestClassify_unknownDefaultsToTransient(t *testing.T) {
	t.Parallel()

	err := errors.New("something unexpected")
	if got := event.Classify(err); got != event.Transient {
		t.Errorf("Classify(unknown) = %v, want %v (Transient)", got, event.Transient)
	}
}

func TestClassify_nilError(t *testing.T) {
	t.Parallel()

	if got := event.Classify(nil); got != event.Rejection {
		t.Errorf("Classify(nil) = %v, want %v (Rejection)", got, event.Rejection)
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		err   error
		retry bool
	}{
		{"transient", event.NewTransient("t", "msg"), true},
		{"nil error", nil, false},
		{"rejection", event.NewRejection("r", "msg"), false},
		{"conflict", event.NewConflict("c", "msg"), false},
		{"corruption", event.NewCorruption("x", "msg"), false},
		{"infrastructure", event.NewInfrastructure("i", "msg"), false},
		{"ErrVersionConflict", event.ErrVersionConflict, false},
		{"ErrDuplicateProjection", event.ErrDuplicateProjection, false},
		{"ErrStoreClosed", event.ErrStoreClosed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := event.IsRetryable(tc.err); got != tc.retry {
				t.Errorf("IsRetryable(%v) = %v, want %v", tc.err, got, tc.retry)
			}
		})
	}
}

func TestError_Format(t *testing.T) {
	t.Parallel()

	err := event.NewTransient("db.timeout", "connection lost")

	got := fmt.Sprintf("%v", err)
	if got != "connection lost" {
		t.Errorf("%%v = %q, want %q", got, "connection lost")
	}

	got = fmt.Sprintf("%s", err)
	if got != "connection lost" {
		t.Errorf("%%s = %q, want %q", got, "connection lost")
	}

	got = fmt.Sprintf("%+v", err)

	want := "transient:db.timeout: connection lost"
	if got != want {
		t.Errorf("%%+v = %q, want %q", got, want)
	}
}

func TestError_Format_withCause(t *testing.T) {
	t.Parallel()

	inner := errors.New("tcp reset")
	err := event.NewTransient("db.timeout", "connection lost").WithCause(inner)

	got := fmt.Sprintf("%+v", err)
	if got == "" {
		t.Error("detailed format should not be empty")
	}

	if !strings.Contains(got, "caused by:") {
		t.Errorf("%%+v with cause should contain 'caused by:', got %q", got)
	}
}

func TestError_Is(t *testing.T) {
	t.Parallel()

	err1 := event.NewRejection("not_found", "user not found")
	err2 := event.NewRejection("not_found", "different message")
	err3 := event.NewConflict("not_found", "user not found")

	if !err1.Is(err2) {
		t.Error("Is should match errors with same Code and Family")
	}

	if err1.Is(err3) {
		t.Error("Is should not match errors with different Family")
	}

	if err1.Is(errors.New("unrelated")) {
		t.Error("Is should not match non-*Error targets")
	}
}

func TestClassify_RegisteredSentinels(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("test.sentinel")
	event.RegisterClassification(sentinel, event.Corruption)

	got := event.Classify(sentinel)
	if got != event.Corruption {
		t.Errorf("Classify(registered) = %v, want Corruption", got)
	}

	wrapped := fmt.Errorf("wrapped: %w", sentinel)

	got = event.Classify(wrapped)

	if got != event.Corruption {
		t.Errorf("Classify(wrapped registered) = %v, want Corruption", got)
	}
}

func TestClassify_CommandQuerySentinels(t *testing.T) {
	t.Parallel()

	if event.Classify(command.ErrHandlerNotFound) != event.Rejection {
		t.Error("ErrHandlerNotFound should be Rejection")
	}

	if event.Classify(command.ErrDispatcherClosed) != event.Infrastructure {
		t.Error("command.ErrDispatcherClosed should be Infrastructure")
	}

	if event.Classify(query.ErrQueryNotSupported) != event.Rejection {
		t.Error("ErrQueryNotSupported should be Rejection")
	}

	if event.Classify(query.ErrDispatcherClosed) != event.Infrastructure {
		t.Error("query.ErrDispatcherClosed should be Infrastructure")
	}
}
