package adttest

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
)

func reproInstance(t *testing.T) {
	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:repro_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}

	db.SetMaxOpenConns(1)

	defer func() { _ = db.Close() }()

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = eng.Close() }()

	ctx := t.Context()

	for _, k := range []string{"k0", "k1"} {
		if err := eng.(metaengine.MapUpdater).MapUpdate(ctx, "col", k, func(any) any { return k }); err != nil {
			t.Fatalf("update %s: %v", k, err)
		}
	}

	res, err := eng.(metaengine.ScanBackend).MapScan(ctx, "col", nil, nil, nil, 10)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	t.Logf("scanned %d", len(res.Items))

	if err := eng.(metaengine.MapBackend).MapDelete(ctx, "col", "k0"); err != nil {
		t.Fatalf("delete after scan: %v", err)
	}
}

func TestReproScanThenDelete(t *testing.T) {
	t.Parallel()

	for i := range 3 {
		t.Run(fmt.Sprintf("inst%d", i), func(t *testing.T) {
			t.Parallel()
			reproInstance(t)
		})
	}
}
