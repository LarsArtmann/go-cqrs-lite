package mysql

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/conformance"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// TestConformance registers the shared suite against the MySQL engine:
// the parity bar every queue engine must clear, third dialect. Live-gated:
// set MYSQL_TEST_DSN to a server DSN (e.g.
// "root@tcp(127.0.0.1:3306)/?parseTime=true"); every subtest gets its
// own throwaway database on that server. Internal test package so the
// Backdate hook can reach the engine's connection.
func TestConformance(t *testing.T) {
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set — skipping MySQL conformance (server DSN, e.g. root@tcp(127.0.0.1:3306)/?parseTime=true)")
	}

	conformance.Run(t, conformance.Harness{
		NewStore: func(t *testing.T) queue.Store[conformance.Payload] {
			t.Helper()

			s, err := Open[conformance.Payload](freshDatabase(t, dsn))
			if err != nil {
				t.Fatalf("open: %v", err)
			}

			return s
		},
		Backdate: backdate,
	})
}

// freshDatabase creates a uniquely-named throwaway database on the
// server behind baseDSN and returns a DSN pointing at it, dropping it
// on cleanup — the "fresh, empty store" the harness contract demands.
func freshDatabase(t *testing.T, baseDSN string) string {
	t.Helper()

	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatalf("database name entropy: %v", err)
	}

	name := "queue_conf_" + hex.EncodeToString(suffix[:])

	server, err := sql.Open("mysql", baseDSN)
	if err != nil {
		t.Fatalf("open server: %v", err)
	}
	defer func() { _ = server.Close() }()

	if _, err := server.Exec("CREATE DATABASE `" + name + "`"); err != nil {
		t.Fatalf("create database: %v", err)
	}

	t.Cleanup(func() {
		if _, err := server.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Errorf("drop database %s: %v", name, err)
		}
	})

	return injectDBName(baseDSN, name)
}

// injectDBName points a server DSN at one database, preserving params.
func injectDBName(dsn, db string) string {
	params := ""
	if i := indexByte(dsn, '?'); i >= 0 {
		dsn, params = dsn[:i], dsn[i:]
	}

	if i := indexByte(dsn, '/'); i >= 0 && !containsAt(dsn, i, "//") {
		dsn = dsn[:i]
	}

	return fmt.Sprintf("%s/%s%s", dsn, db, params)
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}

	return -1
}

func containsAt(s string, i int, sub string) bool {
	return i >= 1 && s[i-1] == '/' && len(sub) > 1
}

// backdate rewinds one task's created_at — the white-box hook the aging
// pins need.
func backdate(t *testing.T, s queue.Store[conformance.Payload], id task.ID, d time.Duration) {
	t.Helper()

	st, ok := s.(*Store[conformance.Payload])
	if !ok {
		t.Fatalf("backdate: store is %T, want *mysql.Store", s)
	}

	if _, err := st.db.Exec(`UPDATE tasks SET created_at = ? WHERE id = ?`,
		time.Now().Add(-d).UnixMilli(), id.String()); err != nil {
		t.Fatalf("backdate %s: %v", id, err)
	}
}
