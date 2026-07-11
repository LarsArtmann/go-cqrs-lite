package turso_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/contracttest"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
)

func TestContract(t *testing.T) {
	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		b, err := turso.New(filepath.Join(t.TempDir(), "contract.db"))
		if err != nil {
			return nil, err
		}

		return b.Bundle, nil
	})
}

func TestMultiDBContract(t *testing.T) {
	contracttest.RunMultiDBSuite(t, func(t *testing.T) (*contracttest.MultiDBTest, error) {
		dir := t.TempDir()

		eventPath := filepath.Join(dir, "events.db")
		queryPath := filepath.Join(dir, "queries.db")
		viewPath := filepath.Join(dir, "views.db")

		b, err := turso.New(
			filepath.Join(dir, "primary.db"),
			turso.WithEventDB(eventPath),
			turso.WithQueryDB(queryPath),
			turso.WithViewDB(viewPath),
		)
		if err != nil {
			return nil, err
		}

		return &contracttest.MultiDBTest{
			Bundle:    b.Bundle,
			EventDSN:  eventPath,
			QueryDSN:  queryPath,
			ViewDSN:   viewPath,
			CountRows: countTursoRows,
		}, nil
	})
}

// TestNewSync_Contract runs the standard contract suite against a NewSync
// bundle. Requires TURSO_SYNC_URL and TURSO_SYNC_TOKEN env vars; skips when
// unset (no remote Turso server available).
func TestNewSync_Contract(t *testing.T) {
	remoteURL := os.Getenv("TURSO_SYNC_URL")
	if remoteURL == "" {
		t.Skip("TURSO_SYNC_URL not set; skipping Turso sync contract tests")
	}

	token := os.Getenv("TURSO_SYNC_TOKEN")

	contracttest.RunSuite(t, func(t *testing.T) (*stack.Bundle, error) {
		b, err := turso.NewSync(
			context.Background(),
			filepath.Join(t.TempDir(), "sync.db"),
			remoteURL,
			token,
		)
		if err != nil {
			return nil, err
		}

		if b.Sync() == nil {
			t.Fatal("Sync() returned nil after NewSync")
		}

		return b.Bundle, nil
	})
}

func countTursoRows(t *testing.T, dbPath, table string) int {
	t.Helper()

	db, err := cqrsturso.Open(cqrsturso.DbPath(dbPath))
	if err != nil {
		t.Fatalf("open %s: %v", filepath.Base(dbPath), err)
	}
	defer func() { _ = db.Close() }()

	var got int

	err = db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&got)
	if err != nil {
		t.Fatalf("count %s.%s: %v", filepath.Base(dbPath), table, err)
	}

	return got
}
