package metaengine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ── Composite key test types ──

type (
	orderLineID string
	productID   string
)

type orderLineCreated struct {
	OrderID   orderLineID
	ProductID productID
	Quantity  int
	Price     float64
}

type orderLineDeleted struct {
	OrderID   orderLineID
	ProductID productID
}

type orderLineView struct {
	OrderID   orderLineID
	ProductID productID
	Quantity  int
	Price     float64
}

type getOrderLine struct {
	OrderID   orderLineID
	ProductID productID
}

func TestInfer_CompositeKey(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getOrderLine, orderLineView]("composite_key",
		metaengine.Infer(orderLineCreated{}, orderLineDeleted{}),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "orderLineCreated", orderLineCreated{
		OrderID: "ord-1", ProductID: "prod-1", Quantity: 3, Price: 9.99,
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if err := store.Apply(ctx, "orderLineCreated", orderLineCreated{
		OrderID: "ord-1", ProductID: "prod-2", Quantity: 1, Price: 4.99,
	}); err != nil {
		t.Fatalf("Apply 2: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getOrderLine, orderLineView](
		ctx, store, getOrderLine{OrderID: "ord-1", ProductID: "prod-1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Quantity != 3 || got.Price != 9.99 {
		t.Errorf("got %+v, want Quantity=3 Price=9.99", got)
	}

	got2, err := metaengine.ExecuteTyped[getOrderLine, orderLineView](
		ctx, store, getOrderLine{OrderID: "ord-1", ProductID: "prod-2"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped 2: %v", err)
	}

	if got2.Quantity != 1 || got2.Price != 4.99 {
		t.Errorf("got %+v, want Quantity=1 Price=4.99", got2)
	}

	// Delete one composite key entry.
	if err := store.Apply(ctx, "orderLineDeleted", orderLineDeleted{
		OrderID: "ord-1", ProductID: "prod-1",
	}); err != nil {
		t.Fatalf("Apply delete: %v", err)
	}

	_, err = metaengine.ExecuteTyped[getOrderLine, orderLineView](
		ctx, store, getOrderLine{OrderID: "ord-1", ProductID: "prod-1"},
	)
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Errorf("after delete: expected ErrNotFound, got %v", err)
	}

	// The other entry should still exist.
	got3, err := metaengine.ExecuteTyped[getOrderLine, orderLineView](
		ctx, store, getOrderLine{OrderID: "ord-1", ProductID: "prod-2"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped 3: %v", err)
	}

	if got3.Quantity != 1 {
		t.Errorf("prod-2 should still exist, got Quantity=%d", got3.Quantity)
	}
}

// ── Filter operator inference test types ──

type scoredItemCreated struct {
	ID    string
	Score int
	Name  string
}

type scoredItemView struct {
	ID    string
	Score int
	Name  string
}

type listScoredItems struct {
	MinScore int
	MaxScore int
}

type scoredItemList struct {
	Items []scoredItemView
}

func TestInfer_FilterOperatorInference(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[listScoredItems, scoredItemList]("filter_ops",
		metaengine.Infer(scoredItemCreated{}),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	items := []scoredItemCreated{
		{ID: "s1", Score: 10, Name: "low"},
		{ID: "s2", Score: 25, Name: "mid"},
		{ID: "s3", Score: 50, Name: "high"},
		{ID: "s4", Score: 80, Name: "top"},
	}

	for _, item := range items {
		if err := store.Apply(ctx, "scoredItemCreated", item); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	// Query with MinScore=20, MaxScore=60 → should return s2 (25) and s3 (50).
	results, err := metaengine.ExecuteTyped[listScoredItems, scoredItemList](
		ctx, store, listScoredItems{MinScore: 20, MaxScore: 60},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if len(results.Items) != 2 {
		t.Fatalf("expected 2 results (25, 50), got %d", len(results.Items))
	}
}

// ── Sort inference test types ──

type eventLogCreated struct {
	ID        string
	Message   string
	CreatedAt time.Time
}

type eventLogView struct {
	ID        string
	Message   string
	CreatedAt time.Time
}

type listEventLog struct {
	Limit int
}

type eventLogList struct {
	Items []eventLogView
}

func TestInfer_SortInference(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[listEventLog, eventLogList]("sort_infer",
		metaengine.Infer(eventLogCreated{}),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 5 {
		if err := store.Apply(ctx, "eventLogCreated", eventLogCreated{
			ID:        "e" + string(rune('1'+i)),
			Message:   "event " + string(rune('1'+i)),
			CreatedAt: base.Add(time.Duration(i) * time.Hour),
		}); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	results, err := metaengine.ExecuteTyped[listEventLog, eventLogList](
		ctx, store, listEventLog{Limit: 3},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	// Sort inference should order by CreatedAt DESC (newest first).
	if len(results.Items) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results.Items))
	}

	if !results.Items[0].CreatedAt.After(results.Items[1].CreatedAt) {
		t.Errorf("expected descending order, got %v then %v",
			results.Items[0].CreatedAt, results.Items[1].CreatedAt)
	}
}

// ── InferFromNamedEvents test types ──

type namedUserCreated struct {
	ID   string
	Name string
}

type namedUserDeleted struct {
	ID string
}

type namedUserView struct {
	ID   string
	Name string
}

type getNamedUser struct {
	ID string
}

func TestInferFromNamedEvents_BasicCRUD(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getNamedUser, namedUserView]("named_infer",
		metaengine.InferFromNamedEvents(
			metaengine.NamedEvent("user.created", namedUserCreated{}),
			metaengine.NamedEvent("user.deleted", namedUserDeleted{}),
		),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Apply using wire event types.
	if err := store.Apply(ctx, "user.created", namedUserCreated{
		ID: "n1", Name: "Named User",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getNamedUser, namedUserView](
		ctx, store, getNamedUser{ID: "n1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "Named User" {
		t.Errorf("Name = %q, want %q", got.Name, "Named User")
	}

	// Delete using wire event type.
	if err := store.Apply(ctx, "user.deleted", namedUserDeleted{ID: "n1"}); err != nil {
		t.Fatalf("Apply delete: %v", err)
	}

	_, err = metaengine.ExecuteTyped[getNamedUser, namedUserView](
		ctx, store, getNamedUser{ID: "n1"},
	)
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Errorf("after delete: expected ErrNotFound, got %v", err)
	}
}

func TestInferFromNamedEvents_EmptyEventTypePanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty event type")
		}
	}()

	_ = metaengine.InferFromNamedEvents(
		metaengine.NamedEvent("", namedUserCreated{}),
	)
}

func TestInferFromNamedEvents_NoSamplesPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for zero samples")
		}
	}()

	_ = metaengine.InferFromNamedEvents()
}

// ── []Struct embedding test types ──

type attachment struct {
	Filename string
	Size     int64
}

type messageWithAttachmentsCreated struct {
	ID          string
	Title       string
	Attachments []attachment
}

type messageWithAttachmentsView struct {
	ID          string
	Title       string
	Attachments []attachment
}

type getMessageWithAttachments struct {
	ID string
}

func TestInfer_SliceOfStructField(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getMessageWithAttachments, messageWithAttachmentsView](
		"slice_struct",
		metaengine.Infer(messageWithAttachmentsCreated{}),
	)

	store, err := metaengine.PlanFromMemory(q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "messageWithAttachmentsCreated", messageWithAttachmentsCreated{
		ID:    "m1",
		Title: "Hello",
		Attachments: []attachment{
			{Filename: "a.txt", Size: 100},
			{Filename: "b.txt", Size: 200},
		},
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getMessageWithAttachments, messageWithAttachmentsView](
		ctx, store, getMessageWithAttachments{ID: "m1"},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Title != "Hello" {
		t.Errorf("Title = %q, want %q", got.Title, "Hello")
	}

	if len(got.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(got.Attachments))
	}

	if got.Attachments[0].Filename != "a.txt" || got.Attachments[1].Size != 200 {
		t.Errorf("attachments not correctly embedded: %+v", got.Attachments)
	}
}
