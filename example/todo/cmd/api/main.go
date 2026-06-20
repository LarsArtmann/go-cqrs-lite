package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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

	app, err := setupApp(ctx, logger, dataDir)
	if err != nil {
		return err
	}

	defer func() { _ = app.bundle.Close() }()

	srv := setupHTTP(logger, app)

	return runServer(ctx, logger, srv)
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
