package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	cqrsmemory "github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/example/todo/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestMux(t *testing.T) *http.ServeMux {
	t.Helper()

	readModelStore, _ := storage.NewMemoryStore()

	eventStore := cqrsmemory.NewMemoryStore()
	eventBus := cqrsmemory.NewMemoryBus()

	todoProjection := projections.NewTodoProjection(readModelStore)

	_ = eventBus.Subscribe(aggregate.EventCreated, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventUpdated, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventStatusChanged, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventCompleted, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventDeleted, todoProjection.Handle)

	cmdDisp := command.NewDispatcher()
	queryDisp := query.NewDispatcher()

	require.NoError(
		t,
		cmdDisp.Register(
			aggregate.CommandCreate,
			commands.NewCreateTodoHandler(eventStore, eventBus).Handle,
		),
	)
	require.NoError(
		t,
		cmdDisp.Register(
			aggregate.CommandUpdate,
			commands.NewUpdateTodoHandler(eventStore, eventBus).Handle,
		),
	)
	require.NoError(
		t,
		cmdDisp.Register(
			aggregate.CommandDelete,
			commands.NewDeleteTodoHandler(eventStore, eventBus).Handle,
		),
	)
	require.NoError(
		t,
		cmdDisp.Register(
			aggregate.CommandChangeStatus,
			commands.NewChangeStatusHandler(eventStore, eventBus).Handle,
		),
	)

	getHandler := queries.NewGetTodoHandler(readModelStore)
	require.NoError(
		t,
		queryDisp.Register(
			queries.GetTodoQueryType,
			func(_ context.Context, q query.Query) (any, error) {
				return getHandler.Handle(q)
			},
		),
	)
	listHandler := queries.NewListTodosHandler(readModelStore)
	require.NoError(
		t,
		queryDisp.Register(
			queries.ListTodosQueryType,
			func(_ context.Context, q query.Query) (any, error) {
				return listHandler.Handle(q)
			},
		),
	)
	countHandler := queries.NewCountTodosHandler(readModelStore)
	require.NoError(
		t,
		queryDisp.Register(
			queries.CountTodosQueryType,
			func(_ context.Context, q query.Query) (any, error) {
				return countHandler.Handle(q)
			},
		),
	)

	mux := http.NewServeMux()
	registerTodoRoutes(mux, cmdDisp, queryDisp)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "healthy"})
	})

	return mux
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "healthy", body["status"])
}

func TestCreateTodo_Success(t *testing.T) {
	t.Parallel()
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := `{"title":"Buy milk","description":"from store","priority":2,"tags":["errands"]}`
	resp, err := http.Post(srv.URL+"/api/v1/todos", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result["id"])
}

func TestCreateTodo_EmptyTitle(t *testing.T) {
	t.Parallel()
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := `{"title":"","description":"no title"}`
	resp, err := http.Post(srv.URL+"/api/v1/todos", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Handler currently returns 500 for all dispatch errors.
	// Wire cqrs-htmx error taxonomy to get proper 400 for domain validation errors.
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestListTodos_Empty(t *testing.T) {
	t.Parallel()
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/todos")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetTodo_NotFound(t *testing.T) {
	t.Parallel()
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/todos/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestFullCommandLifecycle(t *testing.T) {
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	createBody := `{"title":"Test Todo","description":"test desc","priority":1,"tags":["test"]}`
	resp, err := http.Post(
		srv.URL+"/api/v1/todos",
		"application/json",
		strings.NewReader(createBody),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResult map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createResult))
	resp.Body.Close()
	todoID := createResult["id"].(string)
	require.NotEmpty(t, todoID)

	resp, err = http.Get(srv.URL + "/api/v1/todos/" + todoID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	updateBody := `{"title":"Updated Todo","description":"updated"}`
	req, _ := http.NewRequest(
		http.MethodPut,
		srv.URL+"/api/v1/todos/"+todoID,
		strings.NewReader(updateBody),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	statusBody := `{"status":"completed"}`
	req, _ = http.NewRequest(
		http.MethodPatch,
		srv.URL+"/api/v1/todos/"+todoID+"/status",
		strings.NewReader(statusBody),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/todos/"+todoID, nil)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestUpdateTodo_InvalidID(t *testing.T) {
	t.Parallel()
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := `{"title":"Updated","description":"test"}`
	req, _ := http.NewRequest(
		http.MethodPut,
		srv.URL+"/api/v1/todos/invalid-id",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
