package relational

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite" // pure-Go SQLite driver

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func rollupSchema() RelationalSchema {
	return RelationalSchema{Tables: []RelationalTable{
		{
			Name:       "channel_activity_by_day",
			PrimaryKey: []string{"guild_id", "channel_id", "day"},
			Columns: []RelationalColumn{
				{Name: "guild_id", Type: "TEXT"},
				{Name: "channel_id", Type: "TEXT"},
				{Name: "day", Type: "TEXT"},
				{Name: "message_count", Type: "INTEGER"},
			},
		},
		{
			Name:       "attachment_stats",
			PrimaryKey: []string{"guild_id", "day"},
			Columns: []RelationalColumn{
				{Name: "guild_id", Type: "TEXT"},
				{Name: "day", Type: "TEXT"},
				{Name: "total", Type: "INTEGER", Nullable: true},
				{Name: "downloaded", Type: "INTEGER", Nullable: true},
				{Name: "failed", Type: "INTEGER", Nullable: true},
			},
		},
	}}
}

func TestSinkIncrement_NewRow(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "message_count", 1)
	}

	proj, err := NewRelationalProjection("rollup", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "X", nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	assertCounter(t, db, "channel_activity_by_day", "g1", "c1", "2026-07-23", 1)
}

func TestSinkIncrement_ExistingRow(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "message_count", 1)
	}

	proj, err := NewRelationalProjection("rollup", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	for range 3 {
		if err := proj.Handle(ctx, newEvent(t, "X", nil)); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}

	assertCounter(t, db, "channel_activity_by_day", "g1", "c1", "2026-07-23", 3)
	assertCount(t, db, "channel_activity_by_day", 1)
}

func TestSinkIncrement_NegativeDelta(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "message_count", 5)
	}

	proj, err := NewRelationalProjection("rollup", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	_ = proj.Handle(ctx, newEvent(t, "CREATE", nil))

	handlerDec := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "message_count", -2)
	}

	projDec, err := NewRelationalProjection(
		"rollup-dec",
		rollupSchema(),
		db,
		sqlpkg.SQLiteDialect{},
		handlerDec,
		nil,
	)
	if err != nil {
		t.Fatalf("new dec projection: %v", err)
	}

	_ = projDec.Handle(ctx, newEvent(t, "DELETE", nil))

	assertCounter(t, db, "channel_activity_by_day", "g1", "c1", "2026-07-23", 3)
}

func TestSinkIncrement_MultiCounterSameTable(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		key := Row{"guild_id": "g1", "day": "2026-07-23"}

		if err := sink.Increment(ctx, "attachment_stats", key, "total", 1); err != nil {
			return err
		}

		if err := sink.Increment(ctx, "attachment_stats", key, "downloaded", 1); err != nil {
			return err
		}

		return sink.Increment(ctx, "attachment_stats", key, "failed", 1)
	}

	proj, err := NewRelationalProjection("multi", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	for range 2 {
		if err := proj.Handle(ctx, newEvent(t, "ATTACHMENT", nil)); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}

	assertCount(t, db, "attachment_stats", 1)

	var total, downloaded, failed int64

	err = db.QueryRowContext(
		ctx,
		`SELECT total, downloaded, failed FROM attachment_stats WHERE guild_id = ? AND day = ?`,
		"g1", "2026-07-23",
	).Scan(&total, &downloaded, &failed)
	if err != nil {
		t.Fatalf("select stats: %v", err)
	}

	if total != 2 || downloaded != 2 || failed != 2 {
		t.Fatalf("stats = (%d, %d, %d), want (2, 2, 2)", total, downloaded, failed)
	}
}

func TestSinkIncrement_DifferentKeysSeparateRows(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "message_count", 1)
	}

	proj, err := NewRelationalProjection("sep", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	_ = proj.Handle(ctx, newEvent(t, "X", nil))

	handler2 := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c2",
			"day":        "2026-07-23",
		}, "message_count", 1)
	}

	proj2, err := NewRelationalProjection("sep2", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler2, nil)
	if err != nil {
		t.Fatalf("new proj2: %v", err)
	}

	_ = proj2.Handle(ctx, newEvent(t, "X", nil))

	assertCount(t, db, "channel_activity_by_day", 2)
	assertCounter(t, db, "channel_activity_by_day", "g1", "c1", "2026-07-23", 1)
	assertCounter(t, db, "channel_activity_by_day", "g1", "c2", "2026-07-23", 1)
}

func TestSinkIncrement_UnknownTable(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "nonexistent", Row{"id": "x"}, "count", 1)
	}

	proj, err := NewRelationalProjection("err", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(ctx, newEvent(t, "X", nil))
	if !errors.Is(err, errSinkUnknownTable) {
		t.Fatalf("err = %v, want errSinkUnknownTable", err)
	}
}

func TestSinkIncrement_UnknownCounterColumn(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "bogus_column", 1)
	}

	proj, err := NewRelationalProjection("err", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(ctx, newEvent(t, "X", nil))
	if !errors.Is(err, errSinkUnknownColumn) {
		t.Fatalf("err = %v, want errSinkUnknownColumn", err)
	}
}

func TestSinkIncrement_CounterColumnInKey(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":      "g1",
			"channel_id":    "c1",
			"day":           "2026-07-23",
			"message_count": 0,
		}, "message_count", 1)
	}

	proj, err := NewRelationalProjection("err", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(ctx, newEvent(t, "X", nil))
	if !errors.Is(err, errSinkCounterInKey) {
		t.Fatalf("err = %v, want errSinkCounterInKey", err)
	}
}

func TestSinkIncrement_KeyMissingPKColumn(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			// "day" is missing — but it's a PK column
		}, "message_count", 1)
	}

	proj, err := NewRelationalProjection("err", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(ctx, newEvent(t, "X", nil))
	if !errors.Is(err, errSinkKeyMissingPK) {
		t.Fatalf("err = %v, want errSinkKeyMissingPK", err)
	}
}

func TestSinkIncrement_AtomicRollbackOnError(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	wantErr := errors.New("boom after increment")

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "message_count", 1); err != nil {
			return err
		}

		return wantErr
	}

	proj, err := NewRelationalProjection("rb", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(ctx, newEvent(t, "X", nil))
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	assertCount(t, db, "channel_activity_by_day", 0)
}

func TestRelationalProjection_Reset(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	seed := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Increment(ctx, "channel_activity_by_day", Row{
			"guild_id":   "g1",
			"channel_id": "c1",
			"day":        "2026-07-23",
		}, "message_count", 5)
	}

	proj, err := NewRelationalProjection("reset-test", rollupSchema(), db, sqlpkg.SQLiteDialect{},
		seed, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	_ = proj.Handle(ctx, newEvent(t, "X", nil))
	_ = proj.Handle(ctx, newEvent(t, "X", nil))

	assertCount(t, db, "channel_activity_by_day", 1)

	if err := proj.Reset(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}

	assertCount(t, db, "channel_activity_by_day", 0)
	assertCount(t, db, "attachment_stats", 0)

	_ = proj.Handle(ctx, newEvent(t, "X", nil))

	assertCounter(t, db, "channel_activity_by_day", "g1", "c1", "2026-07-23", 5)
}

func assertCounter(
	t *testing.T,
	db *sql.DB,
	table, guildID, channelID, day string,
	want int64,
) {
	t.Helper()

	var got int64

	err := db.QueryRowContext(
		context.Background(),
		`SELECT message_count FROM `+table+
			` WHERE guild_id = ? AND channel_id = ? AND day = ?`,
		guildID, channelID, day,
	).Scan(&got)
	if err != nil {
		t.Fatalf("select counter from %s: %v", table, err)
	}

	if got != want {
		t.Fatalf("%s counter = %d, want %d", table, got, want)
	}
}
