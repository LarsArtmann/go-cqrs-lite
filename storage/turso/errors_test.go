package turso

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestIsQuotaExceeded_NilError(t *testing.T) {
	t.Parallel()

	if IsQuotaExceeded(nil) {
		t.Fatal("IsQuotaExceeded(nil) = true, want false")
	}
}

func TestIsQuotaExceeded_PlainError_NotQuota(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{"network error", errors.New("connection refused")},
		{"DNS error", errors.New("no such host")},
		{"timeout", errors.New("i/o timeout")},
		{
			"sync engine error (non-quota)",
			errors.New("sync engine operation failed: internal error"),
		},
		{
			"database tape error (corruption)",
			errors.New("database tape error: frame checksum mismatch"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if IsQuotaExceeded(tc.err) {
				t.Errorf("IsQuotaExceeded(%q) = true, want false", tc.err.Error())
			}
		})
	}
}

func TestIsQuotaExceeded_QuotaErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{
			"reads blocked (real Turso free-tier 403)",
			errors.New(
				`turso: error: sync engine operation failed: database sync engine error: remote server returned an error: status=403, body={"error":"Operation was blocked: SQL read operations are forbidden (reads are blocked, do you need to upgrade your plan?)"}`,
			),
		},
		{
			"writes blocked",
			errors.New(
				`status=403, body={"error":"Operation was blocked: SQL write operations are forbidden (writes are blocked, do you need to upgrade your plan?)"}`,
			),
		},
		{
			"bare 'Operation was blocked'",
			errors.New("Operation was blocked: too many databases"),
		},
		{
			"bare 'do you need to upgrade your plan'",
			errors.New("storage limit exceeded (do you need to upgrade your plan?)"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !IsQuotaExceeded(tc.err) {
				t.Errorf("IsQuotaExceeded() = false, want true for %q", tc.err.Error())
			}
		})
	}
}

func TestIsQuotaExceeded_WrappedQuotaError(t *testing.T) {
	t.Parallel()

	raw := errors.New(
		`status=403, body={"error":"Operation was blocked: SQL read operations are forbidden (reads are blocked, do you need to upgrade your plan?)"}`,
	)

	wrapped := errorfamily.WrapInfrastructure(
		errorfamily.WrapInfrastructure(raw, "turso.create_sync_db", "NewTursoSyncDb"),
		"storage.open_turso_sync",
		"open turso sync db",
	)

	if !IsQuotaExceeded(wrapped) {
		t.Errorf("IsQuotaExceeded on wrapped chain = false, want true")
	}
}

func TestIsQuotaExceeded_WrappedNonQuotaError(t *testing.T) {
	t.Parallel()

	raw := errors.New("database tape error: frame checksum mismatch")
	wrapped := errorfamily.WrapInfrastructure(raw, "storage.open_turso_sync", "open turso sync db")

	if IsQuotaExceeded(wrapped) {
		t.Errorf("IsQuotaExceeded on wrapped corruption = true, want false")
	}
}

func TestWrapInfraOrOK_NilReturnsNil(t *testing.T) {
	t.Parallel()

	if err := wrapInfraOrOK(nil, "test.code", "test msg"); err != nil {
		t.Fatalf("wrapInfraOrOK(nil, ...) = %v, want nil", err)
	}
}

func TestWrapInfraOrOK_InfrastructureForNormalError(t *testing.T) {
	t.Parallel()

	raw := errors.New("connection refused")
	wrapped := wrapInfraOrOK(raw, "turso.push", "push failed")

	if !errors.Is(wrapped, raw) {
		t.Errorf("wrapped should contain original error: got %v", wrapped)
	}

	var famErr *errorfamily.Error
	if !errors.As(wrapped, &famErr) {
		t.Fatalf("wrapped should be *errorfamily.Error: got %T", wrapped)
	}

	if famErr.ErrorFamily() != errorfamily.Infrastructure {
		t.Errorf("family = %v, want Infrastructure", famErr.ErrorFamily())
	}

	if famErr.ErrorCode() != "turso.push" {
		t.Errorf("code = %q, want 'turso.push'", famErr.ErrorCode())
	}
}

func TestWrapInfraOrOK_RejectionForQuotaError(t *testing.T) {
	t.Parallel()

	raw := errors.New(
		`status=403, body={"error":"Operation was blocked: SQL read operations are forbidden"}`,
	)
	wrapped := wrapInfraOrOK(raw, "turso.push", "push failed")

	if !errors.Is(wrapped, raw) {
		t.Errorf("wrapped should contain original error: got %v", wrapped)
	}

	var famErr *errorfamily.Error
	if !errors.As(wrapped, &famErr) {
		t.Fatalf("wrapped should be *errorfamily.Error: got %T", wrapped)
	}

	if famErr.ErrorFamily() != errorfamily.Rejection {
		t.Errorf("family = %v, want Rejection for quota error", famErr.ErrorFamily())
	}

	if !errors.Is(wrapped, ErrQuotaExceeded) {
		t.Errorf("errors.Is(wrapped, ErrQuotaExceeded) = false, want true")
	}
}

func TestErrQuotaExceeded_MatchesByCodeAndFamily(t *testing.T) {
	t.Parallel()

	// WrapRejection creates a NEW error with the given code — but Error.Is()
	// matches by code+family, so errors.Is should find it.
	wrapped := errorfamily.WrapRejection(
		errors.New("original"),
		"turso.quota_exceeded",
		"some message",
	)

	if !errors.Is(wrapped, ErrQuotaExceeded) {
		t.Errorf("errors.Is should match by code+family: got false")
	}
}
