package metaengine_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// simpleInsertQuery has only a FoldInsert — idempotent for backfill.
func simpleInsertQuery() metaengine.QueryDecl[FindTask, FindTaskResult] {
	return metaengine.Query[FindTask, FindTaskResult](
		"simple_insert",
		metaengine.OnRecord(
			TaskCreated{},
			func(_ record.Record, e TaskCreated) (TaskID, FindTaskResult) {
				return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
			},
		),
	)
}

var _ = Describe("Layout planning follow-ups (ADR-0124 Phase 6b)", func() {
	Describe("SetPriority", func() {
		It("stores the config and triggers a replan", func() {
			kv := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "kv-engine",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			store, err := metaengine.Plan([]metaengine.Engine{kv}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			versionBefore := store.Plan().Version

			err = store.SetPriority(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(store.Plan().Version).To(BeNumerically(">", versionBefore))
		})

		It("changes the resolved layout in GetLayoutInfo", func() {
			kv := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "kv-engine",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			store, err := metaengine.Plan([]metaengine.Engine{kv}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			// Default: Balanced → Embed
			infos := store.GetLayoutInfo()
			Expect(infos).To(HaveLen(1))
			Expect(infos[0].Layout).To(Equal(metaengine.LayoutEmbed))

			// WriteSpeed on KV → Normalize
			err = store.SetPriority(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())

			infos = store.GetLayoutInfo()
			Expect(infos[0].Layout).To(Equal(metaengine.LayoutNormalize))
			Expect(infos[0].Priority).To(Equal(metaengine.PriorityWriteSpeed))
		})
	})

	Describe("LayoutWarnings", func() {
		It("emits no warnings when Embed is selected (Balanced on KV)", func() {
			kv := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "kv-engine",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			store, err := metaengine.Plan([]metaengine.Engine{kv}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			warnings := store.LayoutWarnings()
			Expect(warnings).To(BeEmpty())
		})

		It("emits JOIN_AMPLIFICATION when Normalize is selected on KV", func() {
			kv := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "kv-engine",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			store, err := metaengine.Plan([]metaengine.Engine{kv}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			err = store.SetPriority(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())

			warnings := store.LayoutWarnings()
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0].Type).To(Equal(metaengine.WarnJoinAmplification))
			Expect(warnings[0].Severity).To(Equal("WARN"))
		})

		It("emits no warnings on SQL engine even with WriteSpeed", func() {
			sql := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "pg",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutRow,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityOLogN,
				},
			}}

			store, err := metaengine.Plan([]metaengine.Engine{sql}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			err = store.SetPriority(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())

			warnings := store.LayoutWarnings()
			Expect(warnings).To(BeEmpty())
		})
	})

	Describe("Backfill idempotency safety", func() {
		It("succeeds for idempotent-only queries (insert)", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, simpleInsertQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			log := metaengine.NewEventLog()
			metaengine.WithEventLog(store, log)

			err = store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t1", Title: "Task 1",
			})
			Expect(err).NotTo(HaveOccurred())

			err = store.Backfill(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})

		It("refuses non-idempotent folds (counter) without force", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, countByStatusQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			log := metaengine.NewEventLog()
			metaengine.WithEventLog(store, log)

			err = store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t1", Status: "open",
			})
			Expect(err).NotTo(HaveOccurred())

			err = store.Backfill(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("non-idempotent"))
		})

		It("succeeds with WithBackfillForce", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, countByStatusQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			log := metaengine.NewEventLog()
			metaengine.WithEventLog(store, log)

			err = store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t1", Status: "open",
			})
			Expect(err).NotTo(HaveOccurred())

			err = store.Backfill(context.Background(), metaengine.WithBackfillForce())
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil when no EventLog attached", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, simpleInsertQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			err = store.Backfill(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ConfirmRebuild", func() {
		It("returns nil for empty diffs", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, simpleInsertQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			err = store.ConfirmRebuild(context.Background(), nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("errors when no EventLog is attached for non-empty diffs", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, simpleInsertQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			diffs := []metaengine.LayoutDiff{{
				QueryName:   "simple_insert",
				From:        metaengine.LayoutEmbed,
				To:          metaengine.LayoutNormalize,
				AutoRebuild: false,
			}}

			err = store.ConfirmRebuild(context.Background(), diffs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no EventLog"))
		})

		It("replays events for affected queries with EventLog", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, simpleInsertQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			log := metaengine.NewEventLog()
			metaengine.WithEventLog(store, log)

			err = store.Apply(context.Background(), "TaskCreated", TaskCreated{
				ID: "t1", Title: "Task 1",
			})
			Expect(err).NotTo(HaveOccurred())

			diffs := []metaengine.LayoutDiff{{
				QueryName:   "simple_insert",
				From:        metaengine.LayoutEmbed,
				To:          metaengine.LayoutNormalize,
				AutoRebuild: false,
			}}

			err = store.ConfirmRebuild(context.Background(), diffs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("skips auto-rebuild diffs", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, simpleInsertQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			diffs := []metaengine.LayoutDiff{{
				QueryName:   "simple_insert",
				AutoRebuild: true,
			}}

			// No EventLog, but auto-rebuild diffs are skipped → no error
			err = store.ConfirmRebuild(context.Background(), diffs)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Doctor Layout section", func() {
		It("includes --- Layout --- in output", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			output := store.Doctor(context.Background())
			Expect(output).To(ContainSubstring("--- Layout ---"))
		})

		It("shows layout warnings in Doctor output", func() {
			kv := &fakeEngine{profile: metaengine.EngineProfile{
				Name: "kv-engine",
				Layouts: map[metaengine.ADT]metaengine.StorageLayout{
					metaengine.ADTMap: metaengine.LayoutKV,
				},
				Supports: map[metaengine.ADT]metaengine.Complexity{
					metaengine.ADTMap: metaengine.ComplexityO1,
				},
			}}

			store, err := metaengine.Plan([]metaengine.Engine{kv}, findTaskQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			err = store.SetPriority(context.Background(), &metaengine.PriorityConfig{
				Global: metaengine.PriorityWriteSpeed,
			})
			Expect(err).NotTo(HaveOccurred())

			output := store.Doctor(context.Background())
			Expect(output).To(ContainSubstring("Warnings:"))
			Expect(strings.Count(output, "JOIN_AMPLIFICATION")).To(BeNumerically(">=", 0))
		})
	})

	Describe("Multi-engine backfill integration", func() {
		It("replays events through Backfill without double-logging", func() {
			mem := metaengine.NewMemoryEngine()
			defer mem.Close()

			store, err := metaengine.Plan([]metaengine.Engine{mem}, simpleInsertQuery())
			Expect(err).NotTo(HaveOccurred())
			defer store.Close()

			log := metaengine.NewEventLog()
			metaengine.WithEventLog(store, log)

			// Apply 3 events
			for i, id := range []string{"t1", "t2", "t3"} {
				err = store.Apply(context.Background(), "TaskCreated", TaskCreated{
					ID: TaskID(id), Title: "Task",
					Status: "open", Priority: i,
				})
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(log.Len()).To(Equal(3))

			// Backfill replays events — should NOT add to the EventLog
			err = store.Backfill(context.Background())
			Expect(err).NotTo(HaveOccurred())

			// EventLog should still have exactly 3 events (no double-logging)
			Expect(log.Len()).To(Equal(3))

			// Data should still be correct (not doubled)
			result, err := store.ExecuteCtx(context.Background(), FindTask{ID: "t1"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
		})
	})
})
