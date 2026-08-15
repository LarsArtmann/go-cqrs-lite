package metaengine

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type sharedAttachment struct {
	ID   string
	URL  string
	Size int
}

type sharedNote struct{ Body string }

type sharedTaskResult struct {
	ID          string
	Title       string
	Attachments []sharedAttachment // shared-by-type child
}

type sharedCommentResult struct {
	ID         string
	Attachment *sharedAttachment // shared-by-type child (pointer form)
	Note       sharedNote        // NOT shared — must stay untouched
}

type sharedTaskCreated struct {
	ID   string
	Att  sharedAttachment
	Body string
}

type sharedCommentCreated struct {
	ID  string
	Att sharedAttachment
}

type findSharedTask struct{ ID string }

type findSharedComment struct{ ID string }

func sharedTaskQuery() any {
	return Query[findSharedTask, sharedTaskResult](
		"shared_tasks",
		OnRecord(
			sharedTaskCreated{},
			func(_ record.Record, e sharedTaskCreated) (string, sharedTaskResult) {
				return e.ID, sharedTaskResult{ID: e.ID, Attachments: []sharedAttachment{e.Att}}
			},
		),
	)
}

func sharedCommentQuery() any {
	return Query[findSharedComment, sharedCommentResult](
		"shared_comments",
		OnRecord(
			sharedCommentCreated{},
			func(_ record.Record, e sharedCommentCreated) (string, sharedCommentResult) {
				return e.ID, sharedCommentResult{ID: e.ID, Attachment: &e.Att}
			},
		),
	)
}

// TestWithSharedCollection_ForcesNormalize proves the aggregate-boundary
// opt-in: queries carrying a shared child type are forced to LayoutNormalize
// even when the scoring (ReadSpeed on KV) would embed.
func TestWithSharedCollection_ForcesNormalize(t *testing.T) {
	t.Parallel()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		sharedTaskQuery(), sharedCommentQuery(),
		WithPriorityConfig(&PriorityConfig{Global: PriorityReadSpeed}),
		WithSharedCollection("sharedAttachment"),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	baseline := baselineLayouts(t, sharedTaskQuery(), sharedCommentQuery())

	for _, qa := range store.Plan().Queries {
		if baseline[qa.QueryName] != LayoutEmbed {
			t.Fatalf("test preconditions: %q should default to Embed under ReadSpeed, got %s",
				qa.QueryName, baseline[qa.QueryName])
		}

		if qa.Layout != LayoutNormalize {
			t.Fatalf("query %q should be forced to Normalize, got %s", qa.QueryName, qa.Layout)
		}
	}
}

func baselineLayouts(t *testing.T, queries ...any) map[string]LayoutOption {
	t.Helper()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		append(queries, WithPriorityConfig(&PriorityConfig{Global: PriorityReadSpeed}))...,
	)
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	out := make(map[string]LayoutOption)
	for _, qa := range store.Plan().Queries {
		out[qa.QueryName] = qa.Layout
	}

	return out
}

// TestWithSharedCollection_WarnsWhenSpanningCollections proves the diagnostic:
// a shared type spanning two collections produces a WARN naming both.
func TestWithSharedCollection_WarnsWhenSpanningCollections(t *testing.T) {
	t.Parallel()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		sharedTaskQuery(), sharedCommentQuery(),
		WithSharedCollection("sharedAttachment"),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	var spanningWarn string

	for _, diag := range store.Plan().Diagnostics {
		if diag.Level == DiagLevelWarn && strings.Contains(diag.Message, "sharedAttachment") {
			spanningWarn = diag.Message
		}
	}

	if spanningWarn == "" {
		t.Fatal("expected a WARN for sharedAttachment spanning collections")
	}

	if !strings.Contains(spanningWarn, "shared_tasks") ||
		!strings.Contains(spanningWarn, "shared_comments") {
		t.Fatalf("WARN should name both collections: %q", spanningWarn)
	}
}

// TestWithoutSharedCollection_DefaultLocalChild proves the default: no shared
// declaration → no forced layout, no shared-collection diagnostics.
func TestWithoutSharedCollection_DefaultLocalChild(t *testing.T) {
	t.Parallel()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		sharedTaskQuery(), sharedCommentQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	for _, diag := range store.Plan().Diagnostics {
		if strings.Contains(diag.Message, "shared") && strings.Contains(diag.Message, "Normalize") {
			t.Fatalf(
				"unexpected shared-collection diagnostic without declaration: %q",
				diag.Message,
			)
		}
	}
}

// TestWithSharedCollection_SurvivesReplan proves the declaration is stored on
// the Store and applied by runtime re-plans (AddEngine path).
func TestWithSharedCollection_SurvivesReplan(t *testing.T) {
	t.Parallel()

	store, err := Plan(
		[]Engine{NewMemoryEngine()},
		sharedTaskQuery(),
		WithPriorityConfig(&PriorityConfig{Global: PriorityReadSpeed}),
		WithSharedCollection("sharedAttachment"),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	if err := store.AddEngine(context.Background(), renamed("extra")); err != nil {
		t.Fatal(err)
	}

	if err := store.Replan(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, qa := range store.Plan().Queries {
		if qa.Layout != LayoutNormalize {
			t.Fatalf("query %q lost forced Normalize after replan: %s", qa.QueryName, qa.Layout)
		}
	}
}

// TestSharedTypesInResult_CoversAllFieldShapes locks the reflection contract:
// scalar fields must never panic (regression: unconditional Elem() on string),
// and direct/pointer/slice/map-value child shapes must all match by type name.
func TestSharedTypesInResult_CoversAllFieldShapes(t *testing.T) {
	t.Parallel()

	type mixedResult struct {
		ID         string
		Title      string
		Direct     sharedAttachment
		Ptr        *sharedAttachment
		Slice      []sharedAttachment
		PtrSlice   []*sharedAttachment
		ByMap      map[string]sharedAttachment
		local      sharedNote //nolint:unused // exercises unexported skip
		unexported []sharedAttachment
		Note       sharedNote
	}

	shared := map[string]bool{"sharedAttachment": true}

	got := sharedTypesInResult(reflect.TypeFor[mixedResult](), shared)
	if len(got) != 1 {
		t.Fatalf("expected exactly one distinct match (sharedAttachment), got %v", got)
	}

	if got[0] != "sharedAttachment" {
		t.Fatalf("expected sharedAttachment, got %q", got[0])
	}

	if sharedTypesInResult(reflect.TypeFor[findSharedTask](), shared) != nil {
		t.Fatal("pure-scalar result type must produce no matches and must not panic")
	}
}
