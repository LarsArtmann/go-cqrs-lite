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

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/example/todo/storage"
	"github.com/larsartmann/go-cqrs-lite/memory"
	cqrsPebble "github.com/larsartmann/go-cqrs-lite/pebble"
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

	eventStore := cqrsPebble.NewPebbleStore(readModelStore.DB(), logger)
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

	registerQueryHandlers(queryDisp, readModelStore)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(
			w,
			http.StatusOK,
			map[string]any{"status": "healthy", "timestamp": time.Now().UTC().Unix()},
		)
	})

	registerTodoRoutes(mux, cmdDisp, queryDisp)

	handler := chainMiddleware(
		mux,
		loggingMiddleware(logger),
		corsMiddleware,
	)

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

func registerQueryHandlers(qDisp *query.Dispatcher, store *storage.PebbleStore) {
	_ = query.RegisterTyped(
		qDisp,
		queries.GetTodoQueryType,
		queries.NewGetTodoHandler(store).Handle,
	)
	_ = query.RegisterTyped(
		qDisp,
		queries.ListTodosQueryType,
		queries.NewListTodosHandler(store).Handle,
	)
	_ = query.RegisterTyped(
		qDisp,
		queries.CountTodosQueryType,
		queries.NewCountTodosHandler(store).Handle,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseAggregateID(r *http.Request) (id.AggregateID, error) {
	return id.ParseAggregateID(r.PathValue("id"))
}

func newAggregateID() id.AggregateID {
	return id.NewAggregateID()
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
