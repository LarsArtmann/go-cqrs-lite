package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/example/todo/storage"
	cqrsmemory "github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func setupTestMux(t *testing.T) *http.ServeMux {
	t.Helper()

	readModelStore, _ := storage.NewMemoryStore()

	eventStore := cqrsmemory.NewMemoryStore()
	eventBus := cqrsmemory.NewMemoryBus()

	todoProjection := projections.NewTodoProjection(readModelStore)

	checkpointStore := cqrsmemory.NewMemoryCheckpointStore()
	runner, err := projection.NewRunner(eventStore, eventBus, checkpointStore)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	if err := runner.Register(todoProjection); err != nil {
		t.Fatalf("register projection: %v", err)
	}
	go func() {
		_ = runner.Run(context.Background())
	}()

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
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": statusHealthy})
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

func assertStatus(t *testing.T, resp *http.Response, want int, label string) {
	t.Helper()
	prefix := "Status"
	if label != "" {
		prefix = label + " Status"
	}
	if resp.StatusCode != want {
		t.Fatalf("%s = %d, want %d", prefix, resp.StatusCode, want)
	}
}

func closeBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
}

func doGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

func doPost(ctx context.Context, url, contentType string, body string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	return http.DefaultClient.Do(req)
}

func doRequest(ctx context.Context, method, url string, body string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	ctx := context.Background()
	resp, err := doGet(ctx, srv.URL+"/health")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer closeBody(t, resp)

	assertStatus(t, resp, http.StatusOK, "")

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if body["status"] != statusHealthy {
		t.Errorf("status = %v, want %q", body["status"], statusHealthy)
	}
}

func TestCreateTodo_Success(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	ctx := context.Background()
	body := `{"title":"Buy milk","description":"from store","priority":2,"tags":["errands"]}`
	resp, err := doPost(ctx, srv.URL+"/api/v1/todos", "application/json", body)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	defer closeBody(t, resp)

	assertStatus(t, resp, http.StatusCreated, "")

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

	ctx := context.Background()
	body := `{"title":"","description":"no title"}`
	resp, err := doPost(ctx, srv.URL+"/api/v1/todos", "application/json", body)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	defer closeBody(t, resp)

	assertStatus(t, resp, http.StatusInternalServerError, "")
}

func TestListTodos_Empty(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	ctx := context.Background()
	resp, err := doGet(ctx, srv.URL+"/api/v1/todos")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer closeBody(t, resp)

	assertStatus(t, resp, http.StatusOK, "")
}

func TestGetTodo_NotFound(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	ctx := context.Background()
	resp, err := doGet(ctx, srv.URL+"/api/v1/todos/nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer closeBody(t, resp)

	assertStatus(t, resp, http.StatusBadRequest, "")
}

func TestFullCommandLifecycle(t *testing.T) {
	_, srv := setupTestServer(t)
	ctx := context.Background()

	createBody := `{"title":"Test Todo","description":"test desc","priority":1,"tags":["test"]}`
	resp, err := doPost(ctx, srv.URL+"/api/v1/todos", "application/json", createBody)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	assertStatus(t, resp, http.StatusCreated, "Create")

	var createResult map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	closeBody(t, resp)

	todoID, ok := createResult["id"].(string)
	if !ok || todoID == "" {
		t.Fatal("todoID is empty or wrong type")
	}

	resp, err = doGet(ctx, srv.URL+"/api/v1/todos/"+todoID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertStatus(t, resp, http.StatusOK, "Get")
	closeBody(t, resp)

	updateBody := `{"title":"Updated Todo","description":"updated"}`
	resp, err = doRequest(ctx, http.MethodPut, srv.URL+"/api/v1/todos/"+todoID, updateBody)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	assertStatus(t, resp, http.StatusOK, "Update")
	closeBody(t, resp)

	statusBody := `{"status":"completed"}`
	resp, err = doRequest(
		ctx,
		http.MethodPatch,
		srv.URL+"/api/v1/todos/"+todoID+"/status",
		statusBody,
	)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	assertStatus(t, resp, http.StatusNoContent, "StatusChange")
	closeBody(t, resp)

	resp, err = doRequest(ctx, http.MethodDelete, srv.URL+"/api/v1/todos/"+todoID, "")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertStatus(t, resp, http.StatusNoContent, "Delete")
	closeBody(t, resp)
}

func TestUpdateTodo_InvalidID(t *testing.T) {
	t.Parallel()
	_, srv := setupTestServer(t)

	ctx := context.Background()
	body := `{"title":"Updated","description":"test"}`
	resp, err := doRequest(ctx, http.MethodPut, srv.URL+"/api/v1/todos/invalid-id", body)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer closeBody(t, resp)

	assertStatus(t, resp, http.StatusOK, "")
}
