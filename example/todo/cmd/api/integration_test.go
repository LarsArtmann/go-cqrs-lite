package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/example/todo/storage"
	cqrsmemory "github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/query"
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

	if err := cmdDisp.Register(
		aggregate.CommandCreate,
		commands.NewCreateTodoHandler(eventStore, eventBus).Handle,
	); err != nil {
		t.Fatalf("Register CommandCreate: %v", err)
	}
	if err := cmdDisp.Register(
		aggregate.CommandUpdate,
		commands.NewUpdateTodoHandler(eventStore, eventBus).Handle,
	); err != nil {
		t.Fatalf("Register CommandUpdate: %v", err)
	}
	if err := cmdDisp.Register(
		aggregate.CommandDelete,
		commands.NewDeleteTodoHandler(eventStore, eventBus).Handle,
	); err != nil {
		t.Fatalf("Register CommandDelete: %v", err)
	}
	if err := cmdDisp.Register(
		aggregate.CommandChangeStatus,
		commands.NewChangeStatusHandler(eventStore, eventBus).Handle,
	); err != nil {
		t.Fatalf("Register CommandChangeStatus: %v", err)
	}

	getHandler := queries.NewGetTodoHandler(readModelStore)
	if err := query.RegisterTyped(
		queryDisp,
		queries.GetTodoQueryType,
		getHandler.Handle,
	); err != nil {
		t.Fatalf("Register GetTodo: %v", err)
	}
	listHandler := queries.NewListTodosHandler(readModelStore)
	if err := query.RegisterTyped(
		queryDisp,
		queries.ListTodosQueryType,
		listHandler.Handle,
	); err != nil {
		t.Fatalf("Register ListTodos: %v", err)
	}
	countHandler := queries.NewCountTodosHandler(readModelStore)
	if err := query.RegisterTyped(
		queryDisp,
		queries.CountTodosQueryType,
		countHandler.Handle,
	); err != nil {
		t.Fatalf("Register CountTodos: %v", err)
	}

	mux := http.NewServeMux()
	registerTodoRoutes(mux, cmdDisp, queryDisp)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "healthy"})
	})

	return mux
}

func setupTestServer(t *testing.T) (*http.ServeMux, *httptest.Server) {
	t.Helper()
	mux := setupTestMux(t)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return mux, srv
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("status = %v, want %q", body["status"], "healthy")
	}
}

func TestCreateTodo_Success(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	body := `{"title":"Buy milk","description":"from store","priority":2,"tags":["errands"]}`
	resp, err := http.Post(srv.URL+"/api/v1/todos", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if result["id"] == nil || result["id"] == "" {
		t.Error("id is empty, want non-empty")
	}
}

func TestCreateTodo_EmptyTitle(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	body := `{"title":"","description":"no title"}`
	resp, err := http.Post(srv.URL+"/api/v1/todos", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestListTodos_Empty(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/todos")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestGetTodo_NotFound(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/todos/nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestFullCommandLifecycle(t *testing.T) {
	_, srv := setupTestServer(t)

	createBody := `{"title":"Test Todo","description":"test desc","priority":1,"tags":["test"]}`
	resp, err := http.Post(
		srv.URL+"/api/v1/todos",
		"application/json",
		strings.NewReader(createBody),
	)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create Status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var createResult map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	resp.Body.Close()
	todoID := createResult["id"].(string)
	if todoID == "" {
		t.Fatal("todoID is empty")
	}

	resp, err = http.Get(srv.URL + "/api/v1/todos/" + todoID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	updateBody := `{"title":"Updated Todo","description":"updated"}`
	req, _ := http.NewRequest(
		http.MethodPut,
		srv.URL+"/api/v1/todos/"+todoID,
		strings.NewReader(updateBody),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Update Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	statusBody := `{"status":"completed"}`
	req, _ = http.NewRequest(
		http.MethodPatch,
		srv.URL+"/api/v1/todos/"+todoID+"/status",
		strings.NewReader(statusBody),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("StatusChange Status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/todos/"+todoID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Delete Status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	resp.Body.Close()
}

func TestUpdateTodo_InvalidID(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	body := `{"title":"Updated","description":"test"}`
	req, _ := http.NewRequest(
		http.MethodPut,
		srv.URL+"/api/v1/todos/invalid-id",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}
