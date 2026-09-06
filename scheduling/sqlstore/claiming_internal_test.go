package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestNewClaimingStore_UnknownDialectRejected pins the loud rejection for
// dialects that cannot honor the claim contract. The MySQL path itself is
// proven against a live server by the integration build-tag suite (SKIP
// LOCKED semantics verified on MariaDB 11.4).
func TestNewClaimingStore_UnknownDialectRejected(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "timers.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	_, err = newClaimingStore[struct{}](context.Background(), db, Dialect(42), time.Minute)
	if !errors.Is(err, ErrClaimingUnsupported) {
		t.Fatalf("unknown dialect: got %v, want ErrClaimingUnsupported", err)
	}
}
