package relational

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// capturingConn records the last statement and arguments it was handed and
// returns no rows, so the SQL text can be pinned per dialect without a server.
type capturingConn struct {
	query string
	args  []driver.NamedValue
}

func (c *capturingConn) Prepare(query string) (driver.Stmt, error) { return nil, driver.ErrSkip }

func (c *capturingConn) Close() error { return nil }

func (c *capturingConn) Begin() (driver.Tx, error) { return nil, driver.ErrSkip }

func (c *capturingConn) QueryContext(
	_ context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	c.query, c.args = query, args

	return noRows{}, nil
}

type noRows struct{}

func (noRows) Columns() []string { return []string{"id"} }

func (noRows) Close() error { return nil }

func (noRows) Next([]driver.Value) error { return io.EOF }

// offsetSQL runs an Offset-without-Limit query on a capturing connection.
func offsetSQL(t *testing.T, dialect sqlpkg.Dialect) (string, []any) {
	t.Helper()

	conn := &capturingConn{}
	db := sql.OpenDB(connectorFunc(func(context.Context) (driver.Conn, error) { return conn, nil }))
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewRelationalStore(discordSchema(), db, dialect)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	err = store.Query(context.Background(), "guilds", []string{"id"}, kv.ViewQuery{
		OrderBy: "id", Offset: 3,
	}, func(scan func(dest ...any) error) error { return scan(new(string)) })
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	args := make([]any, 0, len(conn.args))
	for _, a := range conn.args {
		args = append(args, a.Value)
	}

	return conn.query, args
}

type connectorFunc func(context.Context) (driver.Conn, error)

func (f connectorFunc) Connect(ctx context.Context) (driver.Conn, error) { return f(ctx) }

func (f connectorFunc) Driver() driver.Driver { return driverFunc{} }

type driverFunc struct{}

func (driverFunc) Open(string) (driver.Conn, error) { return nil, driver.ErrSkip }

// PostgreSQL rejects a negative LIMIT, so Offset without Limit must not send
// one; SQLite only understands the negative form.
func TestQuery_OffsetWithoutLimit_SQLPerDialect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dialect sqlpkg.Dialect
		wantSQL string
		wantArg []any
	}{
		{
			name:    "postgres",
			dialect: sqlpkg.PostgresDialect{},
			wantSQL: "SELECT id FROM guilds ORDER BY id ASC OFFSET $1",
			wantArg: []any{int64(3)},
		},
		{
			name:    "mysql",
			dialect: sqlpkg.MySQLDialect{},
			wantSQL: "SELECT id FROM guilds ORDER BY `id` ASC LIMIT 18446744073709551615 OFFSET ?",
			wantArg: []any{int64(3)},
		},
		{
			name:    "sqlite",
			dialect: sqlpkg.SQLiteDialect{},
			wantSQL: "SELECT id FROM guilds ORDER BY id ASC LIMIT ? OFFSET ?",
			wantArg: []any{int64(-1), int64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotSQL, gotArg := offsetSQL(t, tt.dialect)

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", gotSQL, tt.wantSQL)
			}

			if strings.Contains(gotSQL, "-1") && tt.name != "sqlite" {
				t.Errorf("%s must not emit a negative LIMIT: %q", tt.name, gotSQL)
			}

			if len(gotArg) != len(tt.wantArg) {
				t.Fatalf("args = %v, want %v", gotArg, tt.wantArg)
			}

			for i := range gotArg {
				if gotArg[i] != tt.wantArg[i] {
					t.Errorf("args = %v, want %v", gotArg, tt.wantArg)
				}
			}
		})
	}
}
