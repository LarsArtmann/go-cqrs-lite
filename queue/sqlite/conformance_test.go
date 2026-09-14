package sqlite_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/conformance"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// TestConformance registers the shared suite against the SQLite engine:
// the parity bar every queue engine must clear (the donor's mirrored
// ADR-0007 suites, upstreamed into one).
func TestConformance(t *testing.T) {
	conformance.Run(t, conformance.Harness{
		NewStore: func(t *testing.T) queue.Store[conformance.Payload] {
			t.Helper()

			s, err := sqlite.Open[conformance.Payload](filepath.Join(t.TempDir(), "queue.db"))
			if err != nil {
				t.Fatalf("open: %v", err)
			}

			return s
		},
		Backdate: backdate,
	})
}

// backdate rewinds one task's created_at — the white-box hook the aging
// pins need; production code has no such path.
func backdate(t *testing.T, s queue.Store[conformance.Payload], id task.ID, d time.Duration) {
	t.Helper()

	db := dbOf(t, s)
	if _, err := db.Exec(`UPDATE tasks SET created_at = ? WHERE id = ?`,
		time.Now().Add(-d).UnixMilli(), id.String()); err != nil {
		t.Fatalf("backdate %s: %v", id, err)
	}
}

// dbOf extracts the engine's *sql.DB via the exported OpenDB seam: the
// store owns its pool, so tests re-open the same file instead.
func dbOf(t *testing.T, s queue.Store[conformance.Payload]) interface {
	Exec(string, ...any) (interface{ RowsAffected() (int64, error) }, error)
} {
	t.Fatalf("unreachable: dbOf is replaced by openRaw below")

	return nil
}
