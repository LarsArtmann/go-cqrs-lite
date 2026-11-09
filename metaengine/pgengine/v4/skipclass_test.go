package pgengine_test

import (
	"errors"
	"strings"
	"testing"
)

// pgSkipClass reports whether err is the server-not-reachable class
// (OQ-10 skip-vs-fail policy): a Postgres server that is not running, not yet
// listening, or unreachable makes live coverage impossible without a defect,
// so the test skips. EVERYTHING ELSE — auth failures, TLS errors,
// unknown-database, schema/migration errors, unexpected server errors —
// fails loudly: those are exactly the classes that once silently deleted
// four ADT subtests from the dgraph suite (2026-09-11, gotchas-testing.md).
//
// art-dupl:accept OQ-10 skip-vs-fail classifier mirrored per engine test package (dgraph precedent); a shared testutil home would drag the CQRS-aware test module into engine test binaries
func pgSkipClass(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	for _, marker := range []string{
		"connection refused",
		"no such host",
		"connection timed out",
		"i/o timeout",
		"dial tcp",
		"failed to connect",
		"name resolver",
		"temporary failure in name resolution",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}

	return false
}

// TestPgSkipClass pins the skip-vs-fail partition: unreachable-class errors
// skip, everything a live server (or the DSN) could get wrong fails loudly.
func TestPgSkipClass(t *testing.T) {
	t.Parallel()

	skipClass := []string{
		"failed to connect to `host=localhost port=5432`: dial error (dial tcp 127.0.0.1:5432: connect: connection refused)",
		"pgengine.New: init: dial tcp: lookup db.invalid: no such host",
		"i/o timeout",
		"read: connection timed out",
		"name resolver: temporary failure in name resolution",
	}

	for _, msg := range skipClass {
		if !pgSkipClass(errors.New(msg)) {
			t.Errorf("pgSkipClass(%q) = false, want true (unreachable class must skip)", msg)
		}
	}

	failLoud := []string{
		"pq: SSL is not enabled on the server",
		"pgengine.New: init: password authentication failed for user \"cqrs\"",
		"x509: certificate signed by unknown authority",
		"pq: database \"nope\" does not exist",
		"pgengine.New: init: relation \"meta_vector\" already exists",
		"",
	}

	for _, msg := range failLoud {
		err := error(nil)
		if msg != "" {
			err = errors.New(msg)
		}

		if pgSkipClass(err) {
			t.Errorf("pgSkipClass(%q) = true, want false (must fail loudly, not skip)", msg)
		}
	}
}
