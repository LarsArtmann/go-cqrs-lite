package event_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

func TestFamily_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		family errorfamily.Family
		want   string
	}{
		{errorfamily.Rejection, "rejection"},
		{errorfamily.Conflict, "conflict"},
		{errorfamily.Transient, "transient"},
		{errorfamily.Corruption, "corruption"},
		{errorfamily.Infrastructure, "infrastructure"},
		{errorfamily.Family(99), "unknown"},
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
		err    *errorfamily.Error
		family errorfamily.Family
		code   string
	}{
		{
			"rejection",
			errorfamily.NewRejection("test.reject", "rejected"),
			errorfamily.Rejection,
			"test.reject",
		},
		{
			"conflict",
			errorfamily.NewConflict("test.conflict", "conflicted"),
			errorfamily.Conflict,
			"test.conflict",
		},
		{
			"transient",
			errorfamily.NewTransient("test.transient", "transient fail"),
			errorfamily.Transient,
			"test.transient",
		},
		{
			"corruption",
			errorfamily.NewCorruption("test.corrupt", "corrupted"),
			errorfamily.Corruption,
			"test.corrupt",
		},
		{
			"infrastructure",
			errorfamily.NewInfrastructure("test.infra", "down"),
			errorfamily.Infrastructure,
			"test.infra",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.err.Family() != tc.family {
				t.Errorf("Family() = %v, want %v", tc.err.Family(), tc.family)
			}

			if tc.err.Code() != tc.code {
				t.Errorf("Code() = %q, want %q", tc.err.Code(), tc.code)
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

	err := errorfamily.NewTransient("test.retry", "retry me").WithCause(inner)
	if !errors.Is(err.Unwrap(), inner) {
		t.Error("WithCause did not set cause")
	}
}

func TestError_WithCause_chaining(t *testing.T) {
	t.Parallel()

	inner := errors.New("root")
	err := errorfamily.NewConflict("test.c", "msg").WithCause(inner)
	wrapped := fmt.Errorf("outer: %w", err)

	extracted, ok := errors.AsType[*errorfamily.Error](wrapped)
	if !ok {
		t.Fatal("errors.AsType should find *errorfamily.Error in wrapped chain")
	}

	if extracted.Code() != "test.c" {
		t.Errorf("Code() = %q, want %q", extracted.Code(), "test.c")
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
		want errorfamily.Family
	}{
		{"version conflict", event.ErrVersionConflict, errorfamily.Conflict},
		{"aggregate not found", event.ErrAggregateNotFound, errorfamily.Rejection},
		{"snapshot not found", snapshot.ErrSnapshotNotFound, errorfamily.Rejection},
		{"store closed", event.ErrStoreClosed, errorfamily.Infrastructure},
		{"bus closed", event.ErrBusClosed, errorfamily.Infrastructure},
		{"snapshot store closed", snapshot.ErrSnapshotStoreClosed, errorfamily.Infrastructure},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := errorfamily.Classify(tc.err); got != tc.want {
				t.Errorf("errorfamily.Classify(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestClassify_typedError(t *testing.T) {
	t.Parallel()

	err := errorfamily.NewRejection("test.r", "bad input")
	if got := errorfamily.Classify(err); got != errorfamily.Rejection {
		t.Errorf("errorfamily.Classify(typed error) = %v, want %v", got, errorfamily.Rejection)
	}
}

func TestClassify_wrappedTypedError(t *testing.T) {
	t.Parallel()

	inner := errorfamily.NewConflict("test.c", "conflict")

	wrapped := fmt.Errorf("handler failed: %w", inner)
	if got := errorfamily.Classify(wrapped); got != errorfamily.Conflict {
		t.Errorf(
			"errorfamily.Classify(wrapped typed error) = %v, want %v",
			got,
			errorfamily.Conflict,
		)
	}
}

func TestClassify_unknownDefaultsToTransient(t *testing.T) {
	t.Parallel()

	err := errors.New("something unexpected")
	if got := errorfamily.Classify(err); got != errorfamily.Transient {
		t.Errorf(
			"errorfamily.Classify(unknown) = %v, want %v (Transient)",
			got,
			errorfamily.Transient,
		)
	}
}

func TestClassify_nilError(t *testing.T) {
	t.Parallel()

	if got := errorfamily.Classify(nil); got != errorfamily.Rejection {
		t.Errorf("errorfamily.Classify(nil) = %v, want %v (Rejection)", got, errorfamily.Rejection)
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		err   error
		retry bool
	}{
		{"transient", errorfamily.NewTransient("t", "msg"), true},
		{"nil error", nil, false},
		{"rejection", errorfamily.NewRejection("r", "msg"), false},
		{"conflict", errorfamily.NewConflict("c", "msg"), false},
		{"corruption", errorfamily.NewCorruption("x", "msg"), false},
		{"infrastructure", errorfamily.NewInfrastructure("i", "msg"), false},
		{"ErrVersionConflict", event.ErrVersionConflict, false},
		{"ErrStoreClosed", event.ErrStoreClosed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := errorfamily.IsRetryable(tc.err); got != tc.retry {
				t.Errorf("errorfamily.IsRetryable(%v) = %v, want %v", tc.err, got, tc.retry)
			}
		})
	}
}

func TestError_Format(t *testing.T) {
	t.Parallel()

	err := errorfamily.NewTransient("db.timeout", "connection lost")

	got := fmt.Sprintf("%v", err)
	want := "[transient:db.timeout] connection lost"
	if got != want {
		t.Errorf("%%v = %q, want %q", got, want)
	}

	got = fmt.Sprintf("%s", err)
	if got != "connection lost" {
		t.Errorf("%%s = %q, want %q", got, "connection lost")
	}

	got = fmt.Sprintf("%+v", err)
	if !strings.Contains(got, "transient") {
		t.Errorf("%%+v should contain family, got %q", got)
	}
	if !strings.Contains(got, "db.timeout") {
		t.Errorf("%%+v should contain code, got %q", got)
	}
	if !strings.Contains(got, "connection lost") {
		t.Errorf("%%+v should contain message, got %q", got)
	}
}

func TestNewf(t *testing.T) {
	t.Parallel()

	err := errorfamily.Newf(errorfamily.Rejection, "test.newf", "formatted %s %d", "hello", 42)
	if err.Code() != "test.newf" {
		t.Errorf("Code() = %q, want %q", err.Code(), "test.newf")
	}
	if !strings.Contains(err.Message(), "formatted hello 42") {
		t.Errorf("Message() = %q, want containing %q", err.Message(), "formatted hello 42")
	}
}

func TestWrapf(t *testing.T) {
	t.Parallel()

	inner := errors.New("root cause")
	err := errorfamily.Wrapf(inner, errorfamily.Transient, "test.wrapf", "wrapped %s", "value")
	if !errors.Is(err, inner) {
		t.Error("Wrapf should preserve cause for errors.Is")
	}
	if err.Code() != "test.wrapf" {
		t.Errorf("Code() = %q, want %q", err.Code(), "test.wrapf")
	}
}

func TestWithContext(t *testing.T) {
	t.Parallel()

	err := errorfamily.NewRejection("test.ctx", "msg")
	result := err.WithContext("key", "value")
	if result.ContextValue("key") != "value" {
		t.Errorf("ContextValue(key) = %q, want %q", result.ContextValue("key"), "value")
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	if got := errorfamily.ExitCode(nil); got != 0 {
		t.Errorf("errorfamily.ExitCode(nil) = %d, want 0", got)
	}
	if got := errorfamily.ExitCode(errorfamily.NewRejection("r", "msg")); got != 1 {
		t.Errorf("errorfamily.ExitCode(Rejection) = %d, want 1", got)
	}
	if got := errorfamily.ExitCode(errorfamily.NewTransient("t", "msg")); got != 75 {
		t.Errorf("errorfamily.ExitCode(Transient) = %d, want 75", got)
	}
}

func TestHandleErrorDetailed(t *testing.T) {
	t.Parallel()

	result := errorfamily.HandleErrorDetailed(nil)
	if result.ExitCode != 0 {
		t.Errorf("HandleErrorDetailed(nil).ExitCode = %d, want 0", result.ExitCode)
	}

	result = errorfamily.HandleErrorDetailed(errorfamily.NewRejection("test.input", "bad input"))
	if result.ExitCode != 1 {
		t.Errorf("HandleErrorDetailed(Rejection).ExitCode = %d, want 1", result.ExitCode)
	}
	if result.Message == "" {
		t.Error("HandleErrorDetailed should produce a non-empty message")
	}
}

func TestError_Format_withCause(t *testing.T) {
	t.Parallel()

	inner := errors.New("tcp reset")
	err := errorfamily.NewTransient("db.timeout", "connection lost").WithCause(inner)

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

	err1 := errorfamily.NewRejection("not_found", "user not found")
	err2 := errorfamily.NewRejection("not_found", "different message")
	err3 := errorfamily.NewConflict("not_found", "user not found")

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
