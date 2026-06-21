package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/example/todo/aggregate"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/projections"
	"github.com/larsartmann/go-cqrs-lite/kv/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	cqrsPebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/v2"
)

type app struct {
	bundle    *stack.Bundle
	cmdDisp   *command.Dispatcher
	queryDisp *query.Dispatcher
	readModel domain.TodoReadModel
}

func setupApp(ctx context.Context, logger *slog.Logger, dataDir string) (*app, error) {
	bundle, err := setupBundle(dataDir)
	if err != nil {
		return nil, err
	}

	rmStore, err := stack.ReadModel[domain.Todo, domain.TodoID](
		bundle, codec.JSONCodec{},
		kv.WithTypedKeyPrefix[domain.Todo, domain.TodoID]("todos:"),
	)
	if err != nil {
		return nil, event.Newf(
			event.Infrastructure,
			"todo.setup.readmodel",
			"create read model store: %v",
			err,
		)
	}

	readModel := newReadModelAdapter(rmStore)

	if err := setupProjection(ctx, bundle, readModel, logger); err != nil {
		return nil, err
	}

	cmdDisp, queryDisp, err := setupDispatchers(bundle, readModel)
	if err != nil {
		return nil, err
	}

	return &app{
		bundle:    bundle,
		cmdDisp:   cmdDisp,
		queryDisp: queryDisp,
		readModel: readModel,
	}, nil
}

func setupBundle(dataDir string) (*stack.Bundle, error) {
	bundle, err := cqrsPebble.New(dataDir)
	if err != nil {
		return nil, event.Newf(event.Infrastructure, "todo.setup.bundle", "create bundle: %v", err)
	}

	return bundle.Bundle, nil
}

func setupProjection(
	_ context.Context,
	bundle *stack.Bundle,
	readModel domain.TodoReadModel,
	_ *slog.Logger,
) error {
	todoProjection := projections.NewTodoProjection(readModel)

	return bundle.Subscriber.SubscribeAll(func(ctx context.Context, evt event.Event) error {
		return todoProjection.Handle(ctx, evt)
	})
}

func setupDispatchers(
	bundle *stack.Bundle,
	readModel domain.TodoReadModel,
) (*command.Dispatcher, *query.Dispatcher, error) {
	cmdDisp := command.NewDispatcher()
	queryDisp := query.NewDispatcher()

	eventStore := bundle.EventSink.(event.Store)

	registerCommand := func(cmdType command.Type, handler command.Handler) {
		_ = cmdDisp.Register(cmdType, handler)
	}

	createH, err := commands.NewCreateTodoHandler(eventStore, bundle.Publisher)
	if err != nil {
		return nil, nil, fmt.Errorf("todo: create handler: %w", err)
	}

	updateH, err := commands.NewUpdateTodoHandler(eventStore, bundle.Publisher)
	if err != nil {
		return nil, nil, fmt.Errorf("todo: update handler: %w", err)
	}

	deleteH, err := commands.NewDeleteTodoHandler(eventStore, bundle.Publisher)
	if err != nil {
		return nil, nil, fmt.Errorf("todo: delete handler: %w", err)
	}

	statusH, err := commands.NewChangeStatusHandler(eventStore, bundle.Publisher)
	if err != nil {
		return nil, nil, fmt.Errorf("todo: status handler: %w", err)
	}

	registerCommand(aggregate.CommandCreate, createH.Handle)
	registerCommand(aggregate.CommandUpdate, updateH.Handle)
	registerCommand(aggregate.CommandDelete, deleteH.Handle)
	registerCommand(aggregate.CommandChangeStatus, statusH.Handle)

	registerQueryHandlers(queryDisp, readModel)

	return cmdDisp, queryDisp, nil
}

func setupHTTP(logger *slog.Logger, app *app) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": statusHealthy})
	})

	registerTodoRoutes(mux, app.cmdDisp, app.queryDisp)

	handler := chainMiddleware(
		mux,
		loggingMiddleware(logger),
		corsMiddleware,
	)

	port := getEnv("PORT", "8080")

	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}

func runServer(ctx context.Context, logger *slog.Logger, srv *http.Server) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)

	go func() {
		logger.Info("Server starting", slog.String("port", srv.Addr))

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return event.Newf(event.Infrastructure, "todo.server.run", "server failed: %v", err)
	case <-quit:
	}

	logger.Info("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return event.Newf(event.Infrastructure, "todo.server.shutdown", "server shutdown: %v", err)
	}

	logger.Info("Server exited")

	return nil
}
