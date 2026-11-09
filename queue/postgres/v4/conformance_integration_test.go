//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/conformance"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

func TestMain(m *testing.M) { pgtestcontainer.TestMain(m) }

// TestConformance registers the shared suite against the Postgres
// engine (SKIP LOCKED claims): the same parity bar the SQLite engine
// clears. Build with -tags integration.
func TestConformance(t *testing.T) {
	conformance.Run(t, conformance.Harness{
		NewStore: func(t *testing.T) queue.Store[conformance.Payload] {
			t.Helper()

			s, err := Open[conformance.Payload](context.Background(), pgtestcontainer.DSN(t), 0)
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
		t.Fatalf("backdate: store is %T, want *postgres.Store", s)
	}

	if _, err := st.pool.Exec(context.Background(),
		`UPDATE tasks SET created_at = $1 WHERE id = $2`,
		time.Now().Add(-d).UnixMilli(), id.String()); err != nil {
		t.Fatalf("backdate %s: %v", id, err)
	}
}
