package bigtableengine_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/bigtable/bttest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestBTProbeFilters(t *testing.T) {
	srv, err := bttest.NewServer("localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	defer srv.Close()

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	ctx := context.Background()

	admin, err := bigtable.NewAdminClient(ctx, "p", "i", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}

	client, err := bigtable.NewClient(ctx, "p", "i", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}

	if err := admin.CreateTable(ctx, "probe2"); err != nil {
		t.Fatal(err)
	}

	if err := admin.CreateColumnFamily(ctx, "probe2", "cqrs"); err != nil {
		t.Fatal(err)
	}

	tbl := client.Open("probe2")

	ts := bigtable.Time(time.Now().Truncate(time.Millisecond))

	mut := bigtable.NewMutation()
	mut.Set("cqrs", "v", ts, []byte(`"v1"`))

	if err := tbl.Apply(ctx, "rk", mut); err != nil {
		t.Fatal(err)
	}

	count := func(name string, opts ...bigtable.ReadOption) {
		row, err := tbl.ReadRow(ctx, "rk", opts...)
		if err != nil {
			t.Logf("%s: ERR %v", name, err)

			return
		}

		t.Logf("%s: %d cells", name, len(row["cqrs"]))
	}

	count("family-only", bigtable.RowFilter(bigtable.FamilyFilter("cqrs")))
	count("latestn-only", bigtable.RowFilter(bigtable.LatestNFilter(1)))
	count("tsrange-only", bigtable.RowFilter(bigtable.TimestampRangeFilterMicros(0, ts+1)))
	count("chain-fam+latest", bigtable.RowFilter(bigtable.ChainFilters(
		bigtable.FamilyFilter("cqrs"), bigtable.LatestNFilter(1))))
	count("chain-latest+ts", bigtable.RowFilter(bigtable.ChainFilters(
		bigtable.LatestNFilter(1), bigtable.TimestampRangeFilterMicros(0, ts+1))))
	count("chain-all3", bigtable.RowFilter(bigtable.ChainFilters(
		bigtable.FamilyFilter("cqrs"), bigtable.LatestNFilter(1),
		bigtable.TimestampRangeFilterMicros(0, ts+1))))
}
