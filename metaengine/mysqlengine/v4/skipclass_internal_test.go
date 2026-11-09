package mysqlengine

import (
	"errors"
	"strings"
	"testing"
)

// mysqlSkipClass reports whether err is the server-not-reachable class
// (OQ-10 skip-vs-fail policy): a MySQL server that is not running, not yet
// listening, or unreachable makes live coverage impossible without a defect,
// so the test skips. EVERYTHING ELSE fails loudly. Internal-test twin of the
// classifier in the mysqlengine_test package (the two test packages cannot
// share unexported helpers).
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

// TestMySQLSkipClassInternal pins the internal classifier's partition.
func TestMySQLSkipClassInternal(t *testing.T) {
	t.Parallel()

	if !mysqlSkipClass(errors.New("dial tcp 127.0.0.1:3306: connect: connection refused")) {
		t.Error("connection refused must classify as skip")
	}

	if mysqlSkipClass(errors.New("Error 1045 (28000): Access denied for user 'cqrs'@'localhost'")) {
		t.Error("auth failure must NOT classify as skip (fail loudly)")
	}

	if mysqlSkipClass(nil) {
		t.Error("nil error must not classify as skip")
	}
}
