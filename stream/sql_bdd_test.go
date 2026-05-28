package stream_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

func newSQLiteTestDB() *sql.DB {
	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	Expect(err).ToNot(HaveOccurred())

	db.SetMaxOpenConns(1)

	return db
}

func makeStreamEvent(eventType event.Type, aggID id.AggregateID, aggType event.AggregateType, version event.Version, opts ...event.Option) event.Event {
	evt, err := event.NewEvent(eventType, aggID, aggType, version, []byte(`{}`), opts...)
	Expect(err).ToNot(HaveOccurred())

	return evt
}

func seedStreamDB(db *sql.DB, tableName string, rows []struct {
	aggType   string
	aggID     string
	version   int
	count     uint
	lastAt    string
	statusInt int
}) {
	for _, r := range rows {
		_, err := db.Exec(fmt.Sprintf(
			`INSERT INTO %s (aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
			 VALUES (?, ?, ?, ?, ?, ?)`, tableName),
			r.aggType, r.aggID, r.version, r.count, r.lastAt, r.statusInt,
		)
		Expect(err).ToNot(HaveOccurred())
	}
}

var _ = Describe("SQL Aggregate Reader", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		db     *sql.DB
		reader *stream.SQLAggregateReader
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		db = newSQLiteTestDB()
		reader = stream.NewSQLAggregateReader(db, "test_")

		_, err := db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			aggregate_type  TEXT NOT NULL,
			aggregate_id    TEXT NOT NULL,
			version         INT  NOT NULL,
			event_count     INT  NOT NULL DEFAULT 0,
			last_event_at   TIMESTAMP NOT NULL,
			tombstone_status INT NOT NULL DEFAULT 0,
			PRIMARY KEY (aggregate_type, aggregate_id)
		)`, "test_stream_aggregates"))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		cancel()
		_ = db.Close()
	})

	Describe("As a developer using the SQL aggregate reader", func() {
		Context("when I list aggregates by type", func() {
			It("should return only aggregates matching the type", func() {
				userID1 := id.NewAggregateID()
				userID2 := id.NewAggregateID()
				orderID := id.NewAggregateID()
				now := time.Now().UTC().Format(time.RFC3339)

				seedStreamDB(db, "test_stream_aggregates", []struct {
					aggType   string
					aggID     string
					version   int
					count     uint
					lastAt    string
					statusInt int
				}{
					{"User", userID1.String(), 3, 3, now, 0},
					{"User", userID2.String(), 1, 1, now, 0},
					{"Order", orderID.String(), 1, 1, now, 0},
				})

				page, err := reader.List(ctx, stream.ListOptions{Type: "User"})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(2))
				Expect(page.HasMore).To(BeFalse())
			})
		})

		Context("when I list without specifying a type", func() {
			It("should return an error", func() {
				_, err := reader.List(ctx, stream.ListOptions{})
				Expect(err).To(MatchError(ContainSubstring("Type is required")))
			})
		})

		Context("when I list with pagination", func() {
			It("should return a limited page and indicate HasMore", func() {
				var ids []string
				now := time.Now().UTC().Format(time.RFC3339)

				for i := 0; i < 5; i++ {
					uid := id.NewAggregateID()
					ids = append(ids, uid.String())
					seedStreamDB(db, "test_stream_aggregates", []struct {
						aggType   string
						aggID     string
						version   int
						count     uint
						lastAt    string
						statusInt int
					}{{"User", uid.String(), 1, 1, now, 0}})
				}

				page, err := reader.List(ctx, stream.ListOptions{Type: "User", Limit: 3})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(3))
				Expect(page.HasMore).To(BeTrue())
			})
		})

		Context("when I paginate with a cursor", func() {
			It("should return items after the cursor", func() {
				var ids []id.AggregateID
				now := time.Now().UTC().Format(time.RFC3339)

				for i := 0; i < 4; i++ {
					uid := id.NewAggregateID()
					ids = append(ids, uid)
					seedStreamDB(db, "test_stream_aggregates", []struct {
						aggType   string
						aggID     string
						version   int
						count     uint
						lastAt    string
						statusInt int
					}{{"User", uid.String(), 1, 1, now, 0}})
				}

				page1, err := reader.List(ctx, stream.ListOptions{Type: "User", Limit: 2})
				Expect(err).ToNot(HaveOccurred())
				Expect(page1.Items).To(HaveLen(2))
				Expect(page1.HasMore).To(BeTrue())

				page2, err := reader.List(ctx, stream.ListOptions{
					Type:  "User",
					Limit: 2,
					After: page1.Items[1].ID,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(page2.Items).To(HaveLen(2))
				Expect(page2.HasMore).To(BeFalse())
			})
		})

		Context("when I list with status filtering", func() {
			BeforeEach(func() {
				now := time.Now().UTC().Format(time.RFC3339)
				activeID := id.NewAggregateID()
				tombstonedID := id.NewAggregateID()

				seedStreamDB(db, "test_stream_aggregates", []struct {
					aggType   string
					aggID     string
					version   int
					count     uint
					lastAt    string
					statusInt int
				}{
					{"User", activeID.String(), 2, 2, now, 0},
					{"User", tombstonedID.String(), 1, 1, now, 1},
				})
			})

			It("should exclude tombstoned aggregates by default", func() {
				page, err := reader.ListWithStatus(ctx, stream.ListOptions{
					Type:      "User",
					Tombstone: stream.TombstoneExclude,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(1))
				Expect(page.Items[0].Status.IsTombstoned()).To(BeFalse())
			})

			It("should include all aggregates with TombstoneInclude", func() {
				page, err := reader.ListWithStatus(ctx, stream.ListOptions{
					Type:      "User",
					Tombstone: stream.TombstoneInclude,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(2))
			})

			It("should show only tombstoned with TombstoneOnly", func() {
				page, err := reader.ListWithStatus(ctx, stream.ListOptions{
					Type:      "User",
					Tombstone: stream.TombstoneOnly,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(1))
				Expect(page.Items[0].Status.IsTombstoned()).To(BeTrue())
			})
		})

		Context("when I list on an empty table", func() {
			It("should return an empty page", func() {
				page, err := reader.List(ctx, stream.ListOptions{Type: "User"})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(BeEmpty())
				Expect(page.HasMore).To(BeFalse())
			})
		})

		Context("when I use List which delegates to ListWithStatus", func() {
			It("should return AggregateRef items without status", func() {
				now := time.Now().UTC().Format(time.RFC3339)
				activeID := id.NewAggregateID()
				tombstonedID := id.NewAggregateID()

				seedStreamDB(db, "test_stream_aggregates", []struct {
					aggType   string
					aggID     string
					version   int
					count     uint
					lastAt    string
					statusInt int
				}{
					{"User", activeID.String(), 1, 1, now, 0},
					{"User", tombstonedID.String(), 1, 1, now, 1},
				})

				page, err := reader.List(ctx, stream.ListOptions{
					Type:      "User",
					Tombstone: stream.TombstoneInclude,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(2))
			})
		})
	})
})

var _ = Describe("Aggregate Projection", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		db     *sql.DB
		proj   *stream.AggregateProjection
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		db = newSQLiteTestDB()

		var err error
		proj, err = stream.NewAggregateProjection(db, "test_")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		cancel()
		_ = db.Close()
	})

	Describe("As a developer building the stream read model", func() {
		Context("when I create a new AggregateProjection", func() {
			It("should create the projection table automatically", func() {
				var count int
				err := db.QueryRow(
					"SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_stream_aggregates'",
				).Scan(&count)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(1))
			})

			It("should return the correct projection name", func() {
				Expect(proj.Name()).To(Equal("stream.aggregate_projection"))
			})

			It("should subscribe to all event types", func() {
				Expect(proj.EventTypes()).To(BeNil())
			})
		})

		Context("when I handle events", func() {
			It("should insert a new aggregate row on first event", func() {
				aggID := id.NewAggregateID()
				evt := makeStreamEvent("user.created", aggID, "User", 1)

				err := proj.Handle(ctx, evt)
				Expect(err).ToNot(HaveOccurred())

				var count int
				err = db.QueryRow("SELECT event_count FROM test_stream_aggregates WHERE aggregate_id = ?",
					aggID.String()).Scan(&count)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(1))
			})

			It("should upsert on subsequent events (increment count, update version)", func() {
				aggID := id.NewAggregateID()

				evt1 := makeStreamEvent("user.created", aggID, "User", 1)
				Expect(proj.Handle(ctx, evt1)).To(Succeed())

				evt2 := makeStreamEvent("user.updated", aggID, "User", 2)
				Expect(proj.Handle(ctx, evt2)).To(Succeed())

				var (
					version int
					count   int
				)
				err := db.QueryRow(
					"SELECT version, event_count FROM test_stream_aggregates WHERE aggregate_id = ?",
					aggID.String(),
				).Scan(&version, &count)
				Expect(err).ToNot(HaveOccurred())
				Expect(version).To(Equal(2))
				Expect(count).To(Equal(2))
			})

			It("should detect and store tombstone status from events", func() {
				aggID := id.NewAggregateID()
				evt := makeStreamEvent("user.deleted", aggID, "User", 1,
					event.WithCustom(event.MetadataKeyTombstone, "true"),
				)

				err := proj.Handle(ctx, evt)
				Expect(err).ToNot(HaveOccurred())

				var status int
				err = db.QueryRow(
					"SELECT tombstone_status FROM test_stream_aggregates WHERE aggregate_id = ?",
					aggID.String(),
				).Scan(&status)
				Expect(err).ToNot(HaveOccurred())
				Expect(event.TombstoneStatus(status).IsTombstoned()).To(BeTrue())
			})

			It("should handle multiple aggregates independently", func() {
				userID := id.NewAggregateID()
				orderID := id.NewAggregateID()

				Expect(proj.Handle(ctx, makeStreamEvent("user.created", userID, "User", 1))).To(Succeed())
				Expect(proj.Handle(ctx, makeStreamEvent("order.placed", orderID, "Order", 1))).To(Succeed())

				var count int
				err := db.QueryRow("SELECT count(*) FROM test_stream_aggregates").Scan(&count)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(2))
			})
		})

		Context("when I create a projection with a duplicate table", func() {
			It("should not fail (CREATE TABLE IF NOT EXISTS)", func() {
				_, err := stream.NewAggregateProjection(db, "test_")
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

var _ = Describe("Stream integration: Projection → SQL Reader pipeline", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		db     *sql.DB
		proj   *stream.AggregateProjection
		reader *stream.SQLAggregateReader
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		db = newSQLiteTestDB()

		var err error
		proj, err = stream.NewAggregateProjection(db, "int_")
		Expect(err).ToNot(HaveOccurred())

		reader = stream.NewSQLAggregateReader(db, "int_")
	})

	AfterEach(func() {
		cancel()
		_ = db.Close()
	})

	Describe("As a developer using the full stream pipeline", func() {
		Context("when I project events and then query", func() {
			It("should list projected aggregates correctly", func() {
				activeID := id.NewAggregateID()
				deletedID := id.NewAggregateID()
				orderID := id.NewAggregateID()

				Expect(proj.Handle(ctx, makeStreamEvent("user.created", activeID, "User", 1,
					event.WithCustom(event.MetadataKeyRebirth, "true"),
				))).To(Succeed())
				Expect(proj.Handle(ctx, makeStreamEvent("user.deleted", deletedID, "User", 1,
					event.WithCustom(event.MetadataKeyTombstone, "true"),
				))).To(Succeed())
				Expect(proj.Handle(ctx, makeStreamEvent("order.placed", orderID, "Order", 1,
					event.WithCustom(event.MetadataKeyRebirth, "true"),
				))).To(Succeed())

				page, err := reader.List(ctx, stream.ListOptions{Type: "User"})
				Expect(err).ToNot(HaveOccurred())
				Expect(page.Items).To(HaveLen(1))
				Expect(page.Items[0].ID).To(Equal(activeID))
			})

			It("should support full pagination through all aggregates", func() {
				for range 5 {
					uid := id.NewAggregateID()
					Expect(proj.Handle(ctx, makeStreamEvent("user.created", uid, "User", 1,
						event.WithCustom(event.MetadataKeyRebirth, "true"),
					))).To(Succeed())
				}

				var allCollected []id.AggregateID
				var cursor id.AggregateID

				for {
					opts := stream.ListOptions{Type: "User", Limit: 2, After: cursor}
					page, err := reader.List(ctx, opts)
					Expect(err).ToNot(HaveOccurred())

					for _, item := range page.Items {
						allCollected = append(allCollected, item.ID)
					}

					if !page.HasMore {
						break
					}

					cursor = page.Items[len(page.Items)-1].ID
				}

				Expect(allCollected).To(HaveLen(5))
			})
		})
	})
})
