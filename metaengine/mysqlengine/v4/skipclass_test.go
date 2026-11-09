package mysqlengine_test

import (
	"errors"
	"strings"
	"testing"
)

// mysqlSkipClass reports whether err is the server-not-reachable class
// (OQ-10 skip-vs-fail policy): a MySQL server that is not running, not yet
// listening, or unreachable makes live coverage impossible without a defect,
// so the test skips. EVERYTHING ELSE — auth failures, unknown-database,
// schema/migration errors, unexpected server errors — fails loudly: those
// are exactly the classes that once silently deleted four ADT subtests from
// the dgraph suite (2026-09-11, gotchas-testing.md).
//
// art-dupl:accept OQ-10 skip-vs-fail classifier mirrored per engine test package (dgraph precedent); a shared testutil home would drag the CQRS-aware test module into engine test binaries
func mysqlSkipClass(err error) bool {
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

// TestMySQLSkipClass pins the skip-vs-fail partition: unreachable-class
// errors skip, everything a live server (or the DSN) could get wrong fails
// loudly.
func TestMySQLSkipClass(t *testing.T) {
	t.Parallel()

	skipClass := []string{
		"mysqlengine.New: init: dial tcp 127.0.0.1:3306: connect: connection refused",
		"mysqlengine.New: init: dial tcp: lookup db.invalid: no such host",
		"i/o timeout",
		"read: connection timed out",
	}

	for _, msg := range skipClass {
		if !mysqlSkipClass(errors.New(msg)) {
			t.Errorf("mysqlSkipClass(%q) = false, want true (unreachable class must skip)", msg)
		}
	}

	failLoud := []string{
		"Error 1045 (28000): Access denied for user 'cqrs'@'localhost' (using password: YES)",
		"Error 1049 (42000): Unknown database 'nope'",
		"mysqlengine.New: invalid DSN: missing the slash separating the database name",
		"x509: certificate signed by unknown authority",
		"mysqlengine.New: init: Table 'meta_vector' already exists",
		"",
	}

	for _, msg := range failLoud {
		err := error(nil)
		if msg != "" {
			err = errors.New(msg)
		}

		if mysqlSkipClass(err) {
			t.Errorf("mysqlSkipClass(%q) = true, want false (must fail loudly, not skip)", msg)
		}
	}
}
