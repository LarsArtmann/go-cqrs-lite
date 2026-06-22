package query_test

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// TestErrorReexports verifies the thin re-export wrappers in errors.go delegate
// correctly to go-error-family with the right family classification, cause
// preservation, and exit-code mapping. These functions are public API surface
// that consumers may depend on, so they must be exercised.
func TestErrorReexports_Constructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(code, msg string) *query.Error
		want query.Family
	}{
		{"NewRejection", query.NewRejection, query.Rejection},
		{"NewConflict", query.NewConflict, query.Conflict},
		{"NewTransient", query.NewTransient, query.Transient},
		{"NewCorruption", query.NewCorruption, query.Corruption},
		{"NewInfrastructure", query.NewInfrastructure, query.Infrastructure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.fn("query.test_code", "test message")
			if err == nil {
				t.Fatalf("%s returned nil", tc.name)
			}

			if err.Family() != tc.want {
				t.Fatalf("Family() = %s, want %s", err.Family(), tc.want)
			}

			if err.Code() != "query.test_code" {
				t.Fatalf("Code() = %q, want %q", err.Code(), "query.test_code")
			}

			if err.Message() != "test message" {
				t.Fatalf("Message() = %q, want %q", err.Message(), "test message")
			}

			if err.Error() == "" {
				t.Fatalf("%s produced empty Error() string", tc.name)
			}

			if got := query.Classify(err); got != tc.want {
				t.Fatalf("Classify(%s) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestErrorReexports_Wrappers(t *testing.T) {
	t.Parallel()

	cause := errors.New("underlying boom")

	tests := []struct {
		name string
		fn   func(err error, code, msg string) *query.Error
		want query.Family
	}{
		{"WrapRejection", query.WrapRejection, query.Rejection},
		{"WrapConflict", query.WrapConflict, query.Conflict},
		{"WrapCorruption", query.WrapCorruption, query.Corruption},
		{"WrapInfrastructure", query.WrapInfrastructure, query.Infrastructure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.fn(cause, "query.wrapped", "wrapper message")
			if err == nil {
				t.Fatalf("%s returned nil", tc.name)
			}

			if err.Family() != tc.want {
				t.Fatalf("Family() = %s, want %s", err.Family(), tc.want)
			}

			if !errors.Is(err, cause) {
				t.Fatalf("%s does not preserve cause via errors.Is", tc.name)
			}

			if err.Unwrap() == nil {
				t.Fatalf("%s.Unwrap() = nil, want cause", tc.name)
			}
		})
	}
}

func TestErrorReexports_GenericWrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("db down")

	err := query.Wrap(cause, query.Transient, "query.timeout", "timed out")
	if err == nil {
		t.Fatal("Wrap returned nil for non-nil cause")
	}

	if err.Family() != query.Transient {
		t.Fatalf("Family() = %s, want Transient", err.Family())
	}

	if !errors.Is(err, cause) {
		t.Fatal("Wrap does not preserve cause")
	}

	// Wrap(nil, ...) returns nil — no error to wrap.
	if got := query.Wrap(nil, query.Rejection, "query.nil", "nothing"); got != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", got)
	}
}

func TestErrorReexports_FormattedConstructors(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk full")

	wrapped := query.Wrapf(cause, query.Infrastructure, "query.io", "failed to write %d bytes", 42)
	if wrapped == nil {
		t.Fatal("Wrapf returned nil")
	}

	if wrapped.Family() != query.Infrastructure {
		t.Fatalf("Wrapf Family() = %s, want Infrastructure", wrapped.Family())
	}

	if wrapped.Message() != "failed to write 42 bytes" {
		t.Fatalf("Wrapf Message() = %q, want formatted message", wrapped.Message())
	}

	if !errors.Is(wrapped, cause) {
		t.Fatal("Wrapf does not preserve cause")
	}

	// Wrapf(nil, ...) returns nil.
	if got := query.Wrapf(nil, query.Rejection, "query.nil", "nothing"); got != nil {
		t.Fatalf("Wrapf(nil) = %v, want nil", got)
	}

	created := query.Newf(query.Conflict, "query.dup", "duplicate %s", "user")
	if created == nil {
		t.Fatal("Newf returned nil")
	}

	if created.Family() != query.Conflict {
		t.Fatalf("Newf Family() = %s, want Conflict", created.Family())
	}

	if created.Message() != "duplicate user" {
		t.Fatalf("Newf Message() = %q, want formatted message", created.Message())
	}
}

func TestErrorReexports_ClassifyAndRetryable(t *testing.T) {
	t.Parallel()

	// Classify recognizes errorfamily errors by their declared family.
	if got := query.Classify(query.NewConflict("c", "m")); got != query.Conflict {
		t.Fatalf("Classify(conflict) = %s, want Conflict", got)
	}

	if got := query.Classify(query.NewTransient("t", "m")); got != query.Transient {
		t.Fatalf("Classify(transient) = %s, want Transient", got)
	}

	// Unknown (non-classified) errors default to Transient (fail-open for retry).
	if got := query.Classify(errors.New("mystery")); got != query.Transient {
		t.Fatalf("Classify(unknown) = %s, want Transient", got)
	}

	// Nil errors classify as Rejection.
	if got := query.Classify(nil); got != query.Rejection {
		t.Fatalf("Classify(nil) = %s, want Rejection", got)
	}

	// IsRetryable is true only for Transient-family errors.
	if !query.IsRetryable(query.NewTransient("t", "m")) {
		t.Fatal("IsRetryable(transient) = false, want true")
	}

	if query.IsRetryable(query.NewRejection("r", "m")) {
		t.Fatal("IsRetryable(rejection) = true, want false")
	}

	if query.IsRetryable(nil) {
		t.Fatal("IsRetryable(nil) = true, want false")
	}
}

func TestErrorReexports_ExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"rejection", query.NewRejection("r", "m"), 1},
		{"conflict", query.NewConflict("c", "m"), 1},
		{"transient", query.NewTransient("t", "m"), 75},
		{"corruption", query.NewCorruption("co", "m"), 65},
		{"infrastructure", query.NewInfrastructure("i", "m"), 69},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := query.ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode(%s) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}
