package systemtest_test

//art-dupl:accept test-fixture twin of system/testdsn_test.go across module boundaries

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
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

// mustApply seeds a fold event and fails the test on error.
func mustApply(t *testing.T, store *metaengine.Store, eventType string, payload any) {
	t.Helper()

	if err := store.Apply(context.Background(), eventType, payload); err != nil {
		t.Fatalf("Apply %s: %v", eventType, err)
	}
}
// loadScaledDeadline mirrors system/load_aware_test.go's load-aware timeout
// helper: tests under a loaded host get a proportional budget instead of a
// fixed deadline (shared box, parallel sessions). Local twin, not testutil:
// mirrors benchkit's loadScaledCeiling.

//art-dupl:accept test-fixture twin of system/load_aware_test.go helper

func loadScaledDeadline(base time.Duration) time.Time {
	return time.Now().Add(time.Duration(currentLoadFactor() * float64(base)))
}

// currentLoadFactor returns the ambient load factor (1-minute load average
// over GOMAXPROCS, clamped to [1, 8]) used to scale wall-clock budgets.
func currentLoadFactor() float64 {
	factor := 1.0

	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		if fields := strings.Fields(string(data)); len(fields) > 0 {
			if load1, err := strconv.ParseFloat(fields[0], 64); err == nil {
				cores := float64(runtime.GOMAXPROCS(0))
				if cores < 1 {
					cores = 1
				}

				factor = load1 / cores
				if factor < 1 {
					factor = 1
				}
				if factor > 8 {
					factor = 8
				}
			}
		}
	}

	return factor
}
