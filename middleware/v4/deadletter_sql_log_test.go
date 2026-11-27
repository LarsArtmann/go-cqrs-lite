package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestSQLDeadLetterStore_HandleLogsInsertFailure(t *testing.T) {
	t.Parallel()

	store := newSQLDeadLetterStore(t)

	var buf bytes.Buffer
	store.logger = slog.New(slog.NewJSONHandler(&buf, nil))

	if err := store.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	store.Handle(context.Background(), DeadLetterEntry{Kind: KindEvent, Type: "test.evt", Attempts: 3})

	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"middleware: dead-letter insert failed; entry not stored"`)) ||
		!bytes.Contains(buf.Bytes(), []byte(`"error":"sql: database is closed"`)) {
		t.Fatalf("insert failure must be logged, got: %q", buf.String())
	}
}
