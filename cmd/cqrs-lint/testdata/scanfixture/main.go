package main

// F031 + F022/F023 typed-path fixture: Scan call sites and Query-declaration
// utilization shapes whose receivers/types have REAL static types
// (packages.Load with NeedTypes). The bufio loop must NOT fire F031 (non-
// metaengine receiver); the TypedReader Scan without WithLimit MUST fire.
// The room-items query carries Volume(100_000) and NO declarative pushdown,
// so Go-side sort (F022) and Go-side filtering (F023) must fire; the
// small-volume and declarative queries are negative controls.

import (
	"bufio"
	"context"
	"os"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// readLines is the false-positive shape from the nsfw-classifier feedback:
// a bufio.Scanner.Scan loop in a metaengine-importing project.
func readLines(f *os.File) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		_ = scanner.Text()
	}
}

// readAll is the coached shape: an unbounded TypedReader Scan.
func readAll(ctx context.Context, r *metaengine.TypedReader[int]) {
	_, _ = r.Scan(ctx)
}

type itemScan struct {
	RoomID string
}

type itemView struct {
	ItemID string
	RoomID string
	Score  float64
}

type smallItemView struct {
	ItemID string
	RoomID string
	Score  float64
}

type declarativeItemView struct {
	ItemID string
	RoomID string
	Score  float64
}

const roomItemsCollection = "room_items"

var roomItemsQuery = metaengine.Query[itemScan, itemView](
	roomItemsCollection,
	metaengine.Volume(100_000),
)

// smallItemsQuery stays below the Volume gate — Go-side filtering is
// legitimately fine and must NOT be coached.
var smallItemsQuery = metaengine.Query[itemScan, smallItemView](
	"small_items",
	metaengine.Volume(100),
)

// declarativeItemsQuery already declares pushdown — never coached.
var declarativeItemsQuery = metaengine.Query[itemScan, declarativeItemView](
	"declarative_items",
	metaengine.Volume(100_000),
	metaengine.FilterOnField[declarativeItemView]("room_id", metaengine.FilterEq),
	metaengine.SortOnField[declarativeItemView]("score", false),
)

// thresholdBadges is the nsfw-classifier grid shape: full read, Go-side
// threshold + room filtering over the room-items read model.
func thresholdBadges(rows []itemView, threshold float64) []itemView {
	out := make([]itemView, 0, len(rows))
	for _, row := range rows {
		if row.Score >= threshold && row.RoomID == "room-1" {
			out = append(out, row)
		}
	}
	return out
}

// sortedByScore is the manual-sort shape over the same read model.
func sortedByScore(rows []itemView) []itemView {
	slices.SortFunc(rows, func(a, b itemView) int {
		if a.Score < b.Score {
			return -1
		}
		return 1
	})
	return rows
}

// smallGrid filters the small-volume query — negative control.
func smallGrid(rows []smallItemView) []smallItemView {
	var out []smallItemView
	for _, row := range rows {
		if row.Score >= 0.5 {
			out = append(out, row)
		}
	}
	return out
}

// declarativeGrid filters the declarative query — negative control.
func declarativeGrid(rows []declarativeItemView) []declarativeItemView {
	var out []declarativeItemView
	for _, row := range rows {
		if row.RoomID == "room-1" {
			out = append(out, row)
		}
	}
	return out
}

func main() {
	readLines(os.Stdin)
	readAll(context.Background(), metaengine.NewReader[int](nil, "items"))
	_ = roomItemsQuery
	_ = smallItemsQuery
	_ = declarativeItemsQuery
}
