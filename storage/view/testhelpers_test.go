package view

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

func openSQLiteInMemory(tb testing.TB) (*sql.DB, error) {
	tb.Helper()
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return nil, fmt.Errorf("generate random DSN suffix: %w", err)
	}
	name := "mem-" + hex.EncodeToString(buf[:])
	return sql.Open("sqlite", fmt.Sprintf(
		"file:%s?mode=memory&cache=shared&_loc=auto&_time_format=sqlite",
		name,
	))
}
