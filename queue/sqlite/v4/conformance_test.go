package sqlite

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/conformance"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// TestConformance registers the shared suite against the SQLite engine:
// the parity bar every queue engine must clear (the donor's mirrored
// ADR-0007 suites, upstreamed into one). Internal test package so the
// Backdate hook can reach the engine's connection — production code has
// no such path.
func TestConformance(t *testing.T) {
	conformance.Run(t, conformance.Harness{
		NewStore: func(t *testing.T) queue.Store[conformance.Payload] {
			t.Helper()

			s, err := Open[conformance.Payload](filepath.Join(t.TempDir(), "queue.db"))
			if err != nil {
				t.Fatalf("open: %v", err)
			}

			return s
		},
		Backdate: backdate,
	})
}

// backdate rewinds one task's created_at — the white-box hook the aging
// pins need.
func backdate(t *testing.T, s queue.Store[conformance.Payload], id task.ID, d time.Duration) {
	t.Helper()

	st, ok := s.(*Store[conformance.Payload])
	if !ok {
		t.Fatalf("backdate: store is %T, want *sqlite.Store", s)
	}

	if _, err := st.db.Exec(`UPDATE tasks SET created_at = ? WHERE id = ?`,
		time.Now().Add(-d).UnixMilli(), id.String()); err != nil {
		t.Fatalf("backdate %s: %v", id, err)
	}
}
