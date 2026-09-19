package adttest

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestProbeSweepShape(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil { t.Fatal(err) }
	db.SetMaxOpenConns(1)
	defer db.Close()
	sq := mustProbeEngine(t, db)
	ctx := t.Context()

	type rec struct {
		Key       string    `json:"key"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	if err := sq.(metaengine.MapUpdater).MapUpdate(ctx, "probeCol", "k1", func(prev any) any {
		return rec{Key: "k1", ExpiresAt: time.UnixMilli(1)}
	}); err != nil { t.Fatal(err) }

	res, err := sq.(metaengine.ScanBackend).MapScan(ctx, "probeCol", func(item any) bool {
		fmt.Printf("ITEM TYPE=%T val=%v\n", item, item)
		return true
	}, nil, nil, 10)
	if err != nil { t.Fatal(err) }
	fmt.Printf("items=%d\n", len(res.Items))
}
