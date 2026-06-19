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

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/readmodel/v2"
	cqrsPebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
)

const (
	statusHealthy     = "healthy"
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 30 * time.Second
)

func main() {
	logger := slog.Default()
	logger.Info("Starting Todo Example — Event Sourcing + CQRS + Projections")

	if err := run(logger); err != nil {
		logger.Error("Fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx := context.Background()
	dataDir := getEnv("DATA_DIR", "./data")

	// One call wires the full CQRS stack: event store, command store, query
	// store, snapshot store, checkpoint store, read-model backend, and event
	// bus — all backed by a single PebbleDB at dataDir.
	bundle, err := cqrsPebble.New(dataDir)
	if err != nil {
		return event.Newf(event.Infrastructure, "todo.cmd.api.main.1", "create bundle: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	// Typed read-model store over the Bundle's shared KV backend.
	rmStore, err := stack.ReadModel[domain.Todo, domain.TodoID](
		bundle, codec.JSONCodec{},
		readmodel.WithKeyPrefix[domain.Todo, domain.TodoID]("todos:"),
	)
	if err != nil {
		return event.Newf(
			event.Infrastructure,
			"todo.cmd.api.main.2",
			"create read model store: %v",
			err,
		)
	}

	readModelStore := newReadModelAdapter(rmStore)

	todoProjection := projections.NewTodoProjection(readModelStore)

	runner, err := projection.NewRunner(bundle.Journal, bundle.Subscriber, bundle.CheckpointStore)
	if err != nil {
		return event.Newf(
			event.Infrastructure,
			"todo.cmd.api.main.3",
			"create projection runner: %v",
			err,
		)
	}

	if err := runner.Register(todoProjection); err != nil {
		return event.Newf(
			event.Infrastructure,
			"todo.cmd.api.main.3",
			"register projection: %v",
			err,
		)
	}

	go func() {
		if runErr := runner.Run(ctx); runErr != nil {
			logger.Error("Projection runner stopped", slog.String("error", runErr.Error()))
		}
	}()

	cmdDisp := command.NewDispatcher()
	queryDisp := query.NewDispatcher()

	// Command handlers need the composite event.Store (Save + Load).
	// The Bundle segregates EventSink/EventSource for ISP, but every preset
	// backs both from the same store, so the assertion always succeeds.
	eventStore := bundle.EventSink.(event.Store)

	if err := cmdDisp.Register(
		aggregate.CommandCreate,
		commands.NewCreateTodoHandler(eventStore, bundle.Publisher).Handle,
	); err != nil {
		return event.Newf(
			event.Infrastructure,
			"todo.cmd.api.main.4",
			"register create command: %v",
			err,
		)
	}

	if err := cmdDisp.Register(
		aggregate.CommandUpdate,
		commands.NewUpdateTodoHandler(eventStore, bundle.Publisher).Handle,
	); err != nil {
		return event.Newf(
			event.Infrastructure,
			"todo.cmd.api.main.5",
			"register update command: %v",
			err,
		)
	}

	if err := cmdDisp.Register(
		aggregate.CommandDelete,
		commands.NewDeleteTodoHandler(eventStore, bundle.Publisher).Handle,
	); err != nil {
		return event.Newf(
			event.Infrastructure,
			"todo.cmd.api.main.6",
			"register delete command: %v",
			err,
		)
	}

	if err := cmdDisp.Register(
		aggregate.CommandChangeStatus,
		commands.NewChangeStatusHandler(eventStore, bundle.Publisher).Handle,
	); err != nil {
		return event.Newf(
			event.Infrastructure,
			"todo.cmd.api.main.7",
			"register change status command: %v",
			err,
		)
	}

	registerQueryHandlers(queryDisp, readModelStore)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(
			w,
			http.StatusOK,
			map[string]string{"status": statusHealthy},
		)
	})

	registerTodoRoutes(mux, cmdDisp, queryDisp)

	handler := chainMiddleware(
		mux,
		loggingMiddleware(logger),
		corsMiddleware,
	)

	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)

	go func() {
		logger.Info("Server starting", slog.String("port", port))

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return event.Newf(event.Infrastructure, "todo.cmd.api.main.8", "server failed: %v", err)
	case <-quit:
	}

	logger.Info("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return event.Newf(event.Infrastructure, "todo.cmd.api.main.9", "server shutdown: %v", err)
	}

	logger.Info("Server exited")

	return nil
}

func registerQueryHandlers(qDisp *query.Dispatcher, store domain.TodoReadModel) {
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

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().Error("failed to encode JSON response", slog.String("error", err.Error()))
	}
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
