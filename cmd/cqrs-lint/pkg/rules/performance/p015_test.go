package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// TP: the ReadAll sits directly inside the RegisterQuery handler closure.
func TestP015_DetectsDirectReadAllInQueryHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

func register(sys *System) {
	system.RegisterQuery[StatsQuery, Stats](sys, qryStats,
		func(ctx context.Context, _ StatsQuery) (Stats, error) {
			journal, ok := events.(Journal)
			if !ok {
				return Stats{}, ErrNoJournal
			}

			all, err := journal.ReadAll(ctx)
			if err != nil {
				return Stats{}, err
			}

			return foldStats(all), nil
		})
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP015Detector(ctx))
	ruletest.AssertRule(t, findings, "P015", 1)
}

// TP: the production shape — the closure delegates to a same-file helper
// (statsFromJournal) that folds the journal.
func TestP015_DetectsOneHopHelperReadAll(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain_queries.go": `package main

func registerCaptureQueries(sys *System) {
	system.RegisterQuery[CaptureStatsQuery, CaptureStats](sys, qryCaptureStats,
		func(ctx context.Context, _ CaptureStatsQuery) (CaptureStats, error) {
			return statsFromJournal(ctx, sys.EventStore())
		})
}

func statsFromJournal(ctx context.Context, events Store) (CaptureStats, error) {
	journal, ok := events.(Journal)
	if !ok {
		return CaptureStats{}, ErrNoJournal
	}

	all, err := journal.ReadAll(ctx)
	if err != nil {
		return CaptureStats{}, err
	}

	return foldStats(all), nil
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP015Detector(ctx))
	ruletest.AssertRule(t, findings, "P015", 1)
}

// NP: a startup one-shot hydrate folds the journal, but no query handler
// reaches it — Open-path reads are not the request-path anti-pattern.
func TestP015_NoFindingForUnreachedHydration(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

func Open(ctx context.Context) *Store {
	s := &Store{}
	s.hydrate(ctx)

	return s
}

func (s *Store) hydrate(ctx context.Context) {
	journal, ok := s.events.(Journal)
	if !ok {
		return
	}

	all, err := journal.ReadAll(ctx)
	if err != nil {
		return
	}

	s.warm(all)
}

func register(sys *System) {
	system.RegisterQuery[StatsQuery, Stats](sys, qryStats,
		func(ctx context.Context, _ StatsQuery) (Stats, error) {
			return Stats{}, nil
		})
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP015Detector(ctx))
	ruletest.AssertRule(t, findings, "P015", 0)
}

// NP: io.ReadAll is package-qualified — never a journal fold.
func TestP015_NoFindingForPackageReadAll(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

func register(sys *System) {
	system.RegisterQuery[DocsQuery, Docs](sys, qryDocs,
		func(ctx context.Context, _ DocsQuery) (Docs, error) {
			body, err := io.ReadAll(r)
			if err != nil {
				return Docs{}, err
			}

			return parseDocs(body), nil
		})
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP015Detector(ctx))
	ruletest.AssertRule(t, findings, "P015", 0)
}

// NP: a non-journal receiver (csv.Reader-style) with a generic name stays
// silent even when reached from a query handler.
func TestP015_NoFindingForNonJournalReceiver(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

func register(sys *System) {
	system.RegisterQuery[RowsQuery, Rows](sys, qryRows,
		func(ctx context.Context, q RowsQuery) (Rows, error) {
			return readRows(q.file)
		})
}

func readRows(path string) (Rows, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	records, err := csvReader(f).ReadAll()
	if err != nil {
		return nil, err
	}

	return Rows(records), nil
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP015Detector(ctx))
	ruletest.AssertRule(t, findings, "P015", 0)
}

// NP: journal folds inside event bus handlers are B019/P001's territory,
// not a query-handler finding.
func TestP015_NoFindingForSubscribeAllHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"projection.go": `package main

func wire(bus *Bus) {
	bus.SubscribeAll(func(ctx context.Context, evt Event) error {
		journal, ok := evt.(Journal)
		if !ok {
			return nil
		}

		_, err := journal.ReadAll(ctx)

		return err
	})
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP015Detector(ctx))
	ruletest.AssertRule(t, findings, "P015", 0)
}
