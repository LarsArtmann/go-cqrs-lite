package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/example/todo/storage"
	"github.com/larsartmann/go-cqrs-lite/memory"
	cqrsStorage "github.com/larsartmann/go-cqrs-lite/storage"
)

func main() {
	logger := slog.Default()
	logger.Info("Starting Todo Example — Event Sourcing + CQRS + Projections")

	ctx := context.Background()
	dataDir := getEnv("DATA_DIR", "./data")

	readModelStore, err := storage.NewPebbleStore(dataDir, logger)
	if err != nil {
		logger.Error("Failed to create read model store", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer readModelStore.Close()

	eventStore := cqrsStorage.NewCQRSAdapter(readModelStore.DB(), logger)
	eventBus := memory.NewMemoryBus()

	todoProjection := projections.NewTodoProjection(readModelStore)

	_ = eventBus.Subscribe(aggregate.EventCreated, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventUpdated, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventStatusChanged, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventCompleted, todoProjection.Handle)
	_ = eventBus.Subscribe(aggregate.EventDeleted, todoProjection.Handle)

	cmdDisp := command.NewDispatcher()
	queryDisp := query.NewDispatcher()

	_ = cmdDisp.Register(
		aggregate.CommandCreate,
		commands.NewCreateTodoHandler(eventStore, eventBus).Handle,
	)
	_ = cmdDisp.Register(
		aggregate.CommandUpdate,
		commands.NewUpdateTodoHandler(eventStore, eventBus).Handle,
	)
	_ = cmdDisp.Register(
		aggregate.CommandDelete,
		commands.NewDeleteTodoHandler(eventStore, eventBus).Handle,
	)
	_ = cmdDisp.Register(
		aggregate.CommandChangeStatus,
		commands.NewChangeStatusHandler(eventStore, eventBus).Handle,
	)

	getHandler := queries.NewGetTodoHandler(readModelStore)
	_ = queryDisp.Register(
		queries.GetTodoQueryType,
		func(_ context.Context, q query.Query) (any, error) {
			return getHandler.Handle(q)
		},
	)
	listHandler := queries.NewListTodosHandler(readModelStore)
	_ = queryDisp.Register(
		queries.ListTodosQueryType,
		func(_ context.Context, q query.Query) (any, error) {
			return listHandler.Handle(q)
		},
	)
	countHandler := queries.NewCountTodosHandler(readModelStore)
	_ = queryDisp.Register(
		queries.CountTodosQueryType,
		func(_ context.Context, q query.Query) (any, error) {
			return countHandler.Handle(q)
		},
	)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(
			w,
			http.StatusOK,
			map[string]any{"status": "healthy", "timestamp": time.Now().UTC().Unix()},
		)
	})

	registerTodoRoutes(mux, cmdDisp, queryDisp)

	handler := cqrshtmx.Chain(
		loggingMiddleware(logger),
		corsMiddleware,
	)(mux)

	port := getEnv("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Server starting", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Info("Server exited")
}

func registerTodoRoutes(
	mux *http.ServeMux,
	cmdDisp *command.Dispatcher,
	queryDisp *query.Dispatcher,
) {
	mux.Handle("GET /api/v1/todos", listTodos(queryDisp))
	mux.Handle("POST /api/v1/todos", createTodo(cmdDisp))
	mux.Handle("GET /api/v1/todos/{id}", getTodo(queryDisp))
	mux.Handle("PUT /api/v1/todos/{id}", updateTodo(cmdDisp))
	mux.Handle("DELETE /api/v1/todos/{id}", deleteTodo(cmdDisp))
	mux.Handle("PATCH /api/v1/todos/{id}/status", changeStatus(cmdDisp))
}

func listTodos(qDisp *query.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, qErr := queries.NewListTodosQuery()
		if qErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": qErr.Error()})
			return
		}
		result, err := qDisp.Dispatch(r.Context(), q)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func createTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Priority    int      `json:"priority"`
			Tags        []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		aggregateID := id.NewAggregateID()
		cmd, cmdErr := commands.NewCreateTodoCommand(
			aggregateID,
			req.Title,
			req.Description,
			req.Priority,
			req.Tags,
		)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}
		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": aggregateID.String()})
	}
}

func getTodo(qDisp *query.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		todoID, err := domain.ParseTodoID(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		q, qErr := queries.NewGetTodoQuery(todoID)
		if qErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": qErr.Error()})
			return
		}
		result, err := qDisp.Dispatch(r.Context(), q)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "todo not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": result})
	}
}

func updateTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := id.ParseAggregateID(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		cmd, cmdErr := commands.NewUpdateTodoCommand(aggregateID, req.Title, req.Description)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}
		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

func deleteTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := id.ParseAggregateID(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		cmd, cmdErr := commands.NewDeleteTodoCommand(aggregateID)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}
		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func changeStatus(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := id.ParseAggregateID(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		var req struct {
			Status domain.TodoStatus `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		cmd, cmdErr := commands.NewChangeStatusCommand(aggregateID, req.Status)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}
		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("HTTP",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Duration("latency", time.Since(start)),
			)
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
