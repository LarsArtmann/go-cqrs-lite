package claiming_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
	_ "modernc.org/sqlite"
)

// timersSpec is the scheduling/sqlstore timers-table shape. The
// byte-exact tests below pin the statements this spec produced BEFORE the
// core was extracted (scheduling/sqlstore/claiming.go, 2026-09-13) —
// proving the extraction changed no emitted SQL.
func timersSpec() claiming.Spec {
	return claiming.Spec{
		Table:       "timers",
		IDColumn:    "id",
		DueColumn:   "fire_at",
		LeaseColumn: "lease_until",
		OrderBy:     "fire_at ASC",
		Returning:   []string{"id", "fire_at", "payload"},
	}
}

func TestPostgresClaimStmt_ByteExact(t *testing.T) {
	t.Parallel()

	const want = `WITH due AS (
SELECT id FROM timers
WHERE fire_at <= $1 AND (lease_until IS NULL OR lease_until <= $1)
ORDER BY fire_at ASC
FOR UPDATE SKIP LOCKED
)
UPDATE timers t SET lease_until = $2 FROM due WHERE t.id = due.id
RETURNING t.id, t.fire_at, t.payload`

	now, until := "2026-09-13T08:00:00Z", "2026-09-13T08:01:00Z"

	got, args := claiming.PostgresClaimStmt(timersSpec(), now, until)

	if got != want {
		t.Errorf("Postgres claim stmt:\ngot:\n%s\nwant:\n%s", got, want)
	}

	if len(args) != 2 || args[0] != now || args[1] != until {
		t.Errorf("Postgres claim args: got %v, want [%s %s]", args, now, until)
	}
}

func TestSQLiteClaimStmt_ByteExact(t *testing.T) {
	t.Parallel()

	const want = `UPDATE timers SET lease_until = ?1
WHERE fire_at <= ?2 AND (lease_until IS NULL OR lease_until <= ?2)
RETURNING id, fire_at, payload`

	now, until := "2026-09-13T08:00:00Z", "2026-09-13T08:01:00Z"

	got, args := claiming.SQLiteClaimStmt(timersSpec(), now, until)

	if got != want {
		t.Errorf("SQLite claim stmt:\ngot:\n%s\nwant:\n%s", got, want)
	}

	if len(args) != 2 || args[0] != until || args[1] != now {
		t.Errorf("SQLite claim args: got %v, want [%s %s]", args, until, now)
	}
}

func TestMySQLClaimSelect_ByteExact(t *testing.T) {
	t.Parallel()

	const want = `SELECT id, fire_at, payload FROM timers
WHERE fire_at <= ? AND (lease_until IS NULL OR lease_until <= ?)
ORDER BY fire_at ASC
FOR UPDATE SKIP LOCKED`

	now := "2026-09-13T08:00:00Z"

	got, args := claiming.MySQLClaimSelect(timersSpec(), now)

	if got != want {
		t.Errorf("MySQL claim select:\ngot:\n%s\nwant:\n%s", got, want)
	}

	if len(args) != 2 || args[0] != now || args[1] != now {
		t.Errorf("MySQL claim args: got %v, want [%s %s]", args, now, now)
	}
}

func TestRenewStmt_ByteExact(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		d    claiming.Dialect
		want string
	}{
		{
			name: "postgres",
			d:    claiming.DialectPostgres,
			want: `UPDATE timers SET lease_until = $1 WHERE id = $2 AND lease_until > $3`,
		},
		{
			name: "mysql",
			d:    claiming.DialectMySQL,
			want: `UPDATE timers SET lease_until = ? WHERE id = ? AND lease_until > ?`,
		},
		{
			name: "sqlite",
			d:    claiming.DialectSQLite,
			want: `UPDATE timers SET lease_until = ?1 WHERE id = ?2 AND lease_until > ?3`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, args := claiming.RenewStmt(tc.d, timersSpec(), "until", "id-1", "now")

			if got != tc.want {
				t.Errorf("renew stmt:\ngot:  %s\nwant: %s", got, tc.want)
			}

			if len(args) != 3 || args[0] != "until" || args[1] != "id-1" || args[2] != "now" {
				t.Errorf("renew args: got %v, want [until id-1 now]", args)
			}
		})
	}
}

func TestSpecKnobs(t *testing.T) {
	t.Parallel()

	s := claiming.Spec{
		Table:       "tasks",
		IDColumn:    "id",
		DueColumn:   "next_visible_at",
		LeaseColumn: "lease_until",
		Returning:   []string{"id", "payload"},
		OrderBy:     "priority DESC, next_visible_at ASC",
	}

	pg, pgArgs := claiming.PostgresClaimStmt(s, "now", "until")

	for _, want := range []string{
		"SELECT id FROM tasks",
		"ORDER BY priority DESC, next_visible_at ASC",
		"UPDATE tasks t SET lease_until = $2",
		"RETURNING t.id, t.payload",
	} {
		if !strings.Contains(pg, want) {
			t.Errorf("Postgres stmt missing %q:\n%s", want, pg)
		}
	}

	if len(pgArgs) != 2 {
		t.Errorf("Postgres args: got %v, want 2", pgArgs)
	}

	lite, _ := claiming.SQLiteClaimStmt(s, "now", "until")

	for _, want := range []string{
		"UPDATE tasks SET lease_until = ?1",
		"RETURNING id, payload",
	} {
		if !strings.Contains(lite, want) {
			t.Errorf("SQLite stmt missing %q:\n%s", want, lite)
		}
	}

	if strings.Contains(lite, "ORDER BY") {
		t.Errorf("SQLite stmt must not order (UPDATE..RETURNING cannot):\n%s", lite)
	}

	my, _ := claiming.MySQLClaimSelect(s, "now")

	if !strings.Contains(my, "ORDER BY priority DESC, next_visible_at ASC") {
		t.Errorf("MySQL select missing custom order:\n%s", my)
	}
}

// TestSQLiteClaimRoundTrip proves the claim semantics end-to-end on a real
// SQLite database: only due+unleased rows are claimed, the lease is
// stamped (a second claim sees nothing), and an expired lease re-opens the
// row (crash reclaim).
func TestSQLiteClaimRoundTrip(t *testing.T) {
	spec := timersSpec()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "timers.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE timers (
id TEXT PRIMARY KEY, fire_at TEXT NOT NULL, payload BLOB NOT NULL)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := claiming.EnsureLeaseColumn(ctx, db, claiming.DialectSQLite, spec); err != nil {
		t.Fatalf("ensure lease column: %v", err)
	}

	// Idempotent: a second call must be a no-op, not an error.
	if err := claiming.EnsureLeaseColumn(ctx, db, claiming.DialectSQLite, spec); err != nil {
		t.Fatalf("ensure lease column (idempotent): %v", err)
	}

	past := time.Date(2026, 9, 13, 7, 0, 0, 0, time.UTC)
	future := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)

	seed := []struct {
		id     string
		fireAt time.Time
	}{
		{"due", past},
		{"future", future},
	}

	for _, row := range seed {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO timers (id, fire_at, payload) VALUES (?, ?, ?)`,
			row.id, row.fireAt.Format(time.RFC3339Nano), []byte("{}"),
		); err != nil {
			t.Fatalf("seed %s: %v", row.id, err)
		}
	}

	claim := func(now, until time.Time) []string {
		t.Helper()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}

		defer func() { _ = tx.Rollback() }()

		query, args := claiming.SQLiteClaimStmt(
			spec,
			now.Format(time.RFC3339Nano),
			until.Format(time.RFC3339Nano),
		)

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			t.Fatalf("claim: %v", err)
		}

		var ids []string

		// rows must close BEFORE the commit, so the deferred close lives in
		// its own scope rather than on the helper itself.
		func() {
			defer func() { _ = rows.Close() }()

			for rows.Next() {
				var id, fireAt string

				var payload []byte

				if err := rows.Scan(&id, &fireAt, &payload); err != nil {
					t.Fatalf("scan: %v", err)
				}

				ids = append(ids, id)
			}

			if err := rows.Err(); err != nil {
				t.Fatalf("rows: %v", err)
			}
		}()

		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}

		return ids
	}

	if got := claim(
		past.Add(time.Minute),
		past.Add(2*time.Minute),
	); len(got) != 1 ||
		got[0] != "due" {
		t.Fatalf("first claim: got %v, want [due]", got)
	}

	// The fresh lease fences: a second claim strictly inside the window sees
	// nothing. (A claim AT the expiry instant re-opens the row: the predicate
	// is lease_until <= now, so equality means expired.)
	if got := claim(past.Add(90*time.Second), past.Add(3*time.Minute)); len(got) != 0 {
		t.Fatalf("second claim inside lease: got %v, want none", got)
	}

	// After the lease expires the row re-opens (crash reclaim).
	if got := claim(
		past.Add(time.Hour),
		past.Add(time.Hour+time.Minute),
	); len(got) != 1 ||
		got[0] != "due" {
		t.Fatalf("claim after lease expiry: got %v, want [due]", got)
	}

	// Neither row is claimable now: the fresh lease from the reclaim above
	// fences "due", and "future" never became due.
	if got := claim(past.Add(time.Hour), past.Add(time.Hour+time.Minute)); len(got) != 0 {
		t.Fatalf("leased and future rows must stay unclaimable: got %v, want none", got)
	}
}

func TestEnsureLeaseColumn_UnknownDialectRejected(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "timers.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	err = claiming.EnsureLeaseColumn(context.Background(), db, claiming.Dialect(42), timersSpec())
	if !errors.Is(err, claiming.ErrUnsupported) {
		t.Fatalf("unknown dialect: got %v, want ErrUnsupported", err)
	}
}
