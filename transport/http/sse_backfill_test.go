package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestSSEAuthMiddleware_RejectsMissingToken(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})

	handler := SSEAuthMiddleware(inner, func(r *http.Request) (SSEClientID, bool) {
		return "", false
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSSEAuthMiddleware_AcceptsValidToken(t *testing.T) {
	t.Parallel()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Query().Get("client") != "user-42" {
			t.Errorf("expected client=user-42, got %q", r.URL.Query().Get("client"))
		}
	})

	handler := SSEAuthMiddleware(inner, func(r *http.Request) (SSEClientID, bool) {
		if r.Header.Get("Authorization") == "Bearer valid-token" {
			return SSEClientID("user-42"), true
		}

		return "", false
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/events", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler should have been called")
	}
}

func TestBackfillHandler_ReturnsEvents(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Test", aggID)

	evt0, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{"n":0}`))
	evt1, _ := event.NewEvent("test.event", aggID, "Test", 2, []byte(`{"n":1}`))
	evt2, _ := event.NewEvent("test.event", aggID, "Test", 3, []byte(`{"n":2}`))

	_ = store.Save(context.Background(), ref, []event.Event{evt0, evt1, evt2}, 0)

	handler := BackfillHandler(store)

	// Request events after evt0.
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/backfill?after="+evt0.ID().String()+"&limit=10",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	type item struct {
		ID      string          `json:"id"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	var items []item
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 events, got %d", len(items))
	}

	if items[0].ID != evt1.ID().String() {
		t.Errorf("expected first event ID %s, got %s", evt1.ID().String(), items[0].ID)
	}
}

func TestBackfillHandler_MissingAfterParam(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	handler := BackfillHandler(store)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/backfill", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestBackfillHandler_LimitsTo1000(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Test", aggID)
	evt0, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{}`))
	_ = store.Save(context.Background(), ref, []event.Event{evt0}, 0)

	handler := BackfillHandler(store)

	// limit=99999 should be silently capped to 1000, not error.
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/backfill?after="+evt0.ID().String()+"&limit=99999",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestBackfillHandlerWithTransform_AppliesTransform(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Test", aggID)

	evt0, _ := event.NewEvent("test.event", aggID, "Test", 1, []byte(`{"raw":true}`))
	evt1, _ := event.NewEvent("test.event", aggID, "Test", 2, []byte(`{"seq":1}`))

	_ = store.Save(context.Background(), ref, []event.Event{evt0, evt1}, 0)

	handler := BackfillHandlerWithTransform(store, func(evt event.Event) []byte {
		return []byte(`{"transformed":true}`)
	})

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/backfill?after="+evt0.ID().String()+"&limit=10",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()

	if !strings.Contains(body, `{"transformed":true}`) {
		t.Errorf("transformed payload missing; body: %q", body)
	}

	if strings.Contains(body, `{"raw":true}`) {
		t.Errorf("raw payload should NOT appear; body: %q", body)
	}
}
