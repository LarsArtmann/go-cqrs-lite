package mysql

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/conformance"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
	"github.com/larsartmann/go-cqrs-lite/testutil/mysqltestcontainer/v4"
)

// TestMain resolves the DSN by priority: MYSQL_TEST_DSN (CI / nix legs)
// > a local MariaDB testcontainer > skip. See mysqltestcontainer.TestMain.
func TestMain(m *testing.M) { mysqltestcontainer.TestMain(m) }

// TestConformance registers the shared suite against the MySQL engine:
// the parity bar every queue engine must clear, third dialect. Every
// subtest gets its own throwaway database on the resolved server.
// Internal test package so the Backdate hook can reach the engine's
// connection.
func TestConformance(t *testing.T) {
	dsn := mysqltestcontainer.DSN(t)

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

	if _, err := server.Exec("CREATE DATABASE `" + name + "`"); err != nil {
		t.Fatalf("create database: %v", err)
	}

	// Drop and close in ONE cleanup (LIFO order matters: dropping needs
	// the live handle).
	t.Cleanup(func() {
		if _, err := server.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Errorf("drop database %s: %v", name, err)
		}

		_ = server.Close()
	})

	dbDSN, err := injectDBName(baseDSN, name)
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	return dbDSN
}

// injectDBName points a server DSN at one database, preserving every
// other component (parsed and re-rendered by the driver itself).
func injectDBName(dsn, db string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}

	cfg.DBName = db

	return cfg.FormatDSN(), nil
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
