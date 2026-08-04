package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

// TestP012_P013_DiscordSyncProductionCode verifies that both P012 and P013
// produce ZERO findings when run against DiscordSync's actual internal/db/db.go
// source code, which sets busy_timeout and journal_mode(WAL) via a multi-line
// const concatenated to the DSN string. This is the regression test for the
// false positive that caused DiscordSync to disable P012/P013 in .cqrs-lint.json.
func TestP012_P013_DiscordSyncProductionCode(t *testing.T) {
	t.Parallel()

	// This is the actual source from DiscordSync internal/db/db.go, slightly
	// trimmed to the relevant open function and const (imports and unrelated
	// methods removed for test conciseness, but the DSN construction pattern
	// is verbatim from production).
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package db

import (
	"database/sql"
	"os"
	"path/filepath"
	_ "modernc.org/sqlite"
)

const defaultDirPerm = 0o750

const (
	sqliteMaxOpenConns = 1
	sqliteMaxIdleConns = 1
)

const sqlitePragmaDSNParams = "?_pragma=busy_timeout(15000)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=wal_autocheckpoint(4000)" +
	"&_pragma=cache_size(-65536)" +
	"&_pragma=temp_store(MEMORY)" +
	"&_pragma=mmap_size(268435456)"

func openSQLite(path string) (*DB, error) {
	dir := filepath.Dir(path)

	err := os.MkdirAll(dir, defaultDirPerm)
	if err != nil {
		return nil, err
	}

	dsn := path + sqlitePragmaDSNParams

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(sqliteMaxOpenConns)
	database.SetMaxIdleConns(sqliteMaxIdleConns)

	return &DB{DB: database}, nil
}

type DB struct {
	*sql.DB
}
`,
	})

	p012Findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, p012Findings, "P012", 0)

	p013Findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, p013Findings, "P013", 0)
}

// TestP013_DiscordSyncMigrateGoPostOpenPragma verifies that a file setting
// busy_timeout via PRAGMA in Migrate() (not DSN) is also not flagged.
// This mirrors DiscordSync's internal/db/migrate.go pattern.
func TestP013_DiscordSyncMigrateGoPostOpenPragma(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"migrate.go": `package db

import (
	"context"
	"database/sql"
)

func migrateDB(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		"PRAGMA busy_timeout = 15000",
		"PRAGMA journal_mode = WAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}
	return nil
}
`,
	})

	// migrate.go doesn't call sql.Open, so there should be no findings at all.
	p013Findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, p013Findings, "P013", 0)
}
