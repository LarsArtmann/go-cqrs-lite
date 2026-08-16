package system_test

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// testDSNSeq makes SQLite test DSNs unique across -count replays, which run
// in one process where t.Name() alone would repeat.
var testDSNSeq atomic.Int64

// sqliteTestDSN returns a shared-cache in-memory SQLite DSN unique per
// invocation. Keying only on t.Name() makes repeated runs (-count>1) share
// one database, so journal rows accumulate across replays.
func sqliteTestDSN(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), testDSNSeq.Add(1))
}

// sqliteFileDSN returns a file-backed SQLite DSN under t.TempDir(). Unlike
// sqliteTestDSN, the database genuinely survives Close/reopen — the fixture
// for restart/replay tests. Shared-cache in-memory databases are destroyed
// when the last connection closes, and engines now own (and close) their
// self-opened *sql.DB, so only a file persists a full system.Close().
func sqliteFileDSN(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "journal.db")
}
