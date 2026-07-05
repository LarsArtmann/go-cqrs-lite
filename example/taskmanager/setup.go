package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// Server is the composition root — all wired components live here.
type Server struct {
	Bundle       *stack.Bundle
	Repo         *decider.Repository[TaskState]
	CmdDisp      *command.Dispatcher
	ReadModel    *kv.TypedStore[TaskView, TaskID]
	Mat          *stack.Materialize[TaskView, TaskID]
	CatchUp      *cqrswatermill.CatchUpSubscriber
	Logger       *slog.Logger
	otelProvider *cqrsotel.Provider
	signer       signing.Signer
	httpServer   *http.Server
}

// Config configures the Server.
type Config struct {
	DatabasePath string
	HTTPAddr     string
}

func DefaultConfig() Config {
	return Config{DatabasePath: ":memory:", HTTPAddr: ":8080"}
}

// NewServer wires all components and returns a ready-to-start Server.
//
// The deployer picks infrastructure (one line: sqlite.New). The consumer
// code (domain.go, events.go, decider.go, projection.go) is identical
// regardless of database choice.
func NewServer(cfg Config, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	bundle, err := sqlite.New(
		cfg.DatabasePath,
		sqlite.WithOptimizations(),
	)
	if err != nil {
		return nil, fmt.Errorf("setup: sqlite bundle: %w", err)
	}

	// SQLite's connection pool can create separate databases for :memory:.
	// Restricting to a single connection ensures schema visibility.
	if db, ok := bundle.Database().(*sql.DB); ok {
		db.SetMaxOpenConns(1)
	}

	repo, err := stack.Repository(bundle, TaskDecider)
	if err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("setup: repository: %w", err)
	}

	rmStore, err := stack.ReadModel[TaskView, TaskID](bundle, bundle.DefaultCodec())
	if err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("setup: read model: %w", err)
	}

	mat, err := stack.NewMaterialize[TaskView, TaskID](bundle, bundle.DefaultCodec(), taskViewKey)
	if err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("setup: materialize: %w", err)
	}

	configureProjection(mat)

	catchUp, err := bundle.CatchUpSubscriber()
	if err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("setup: catch-up subscriber: %w", err)
	}

	srv := &Server{
		Bundle:    bundle,
		Repo:      repo,
		CmdDisp:   command.NewDispatcher(),
		ReadModel: rmStore,
		Mat:       mat,
		CatchUp:   catchUp,
		Logger:    logger,
	}

	registerHandlers(srv)

	if err := setupFeatures(srv); err != nil {
		_ = srv.Bundle.Close()

		return nil, fmt.Errorf("setup: features: %w", err)
	}

	return srv, nil
}

// Start launches the projection goroutine. Call StartHTTP separately
// for the HTTP API (not needed in tests that use httptest).
func (s *Server) Start(ctx context.Context) error {
	msgs, err := s.CatchUp.Subscribe(ctx, cqrswatermill.DefaultEventBusTopic)
	if err != nil {
		return fmt.Errorf("start: subscribe: %w", err)
	}

	handler := s.Mat.HandlerFunc()

	go func() {
		for msg := range msgs {
			if err := handler(msg); err != nil {
				s.Logger.Error("projection error",
					"type", msg.Metadata.Get("event_type"), "error", err)
			}

			msg.Ack()
		}
	}()

	return nil
}

// StartHTTP launches the HTTP API server. Call after Start.
func (s *Server) StartHTTP(addr string) error {
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		s.Logger.Info("HTTP server starting", "addr", s.httpServer.Addr)

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.Logger.Error("HTTP server", "error", err)
		}
	}()

	return nil
}

// Stop gracefully drains all components.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.httpServer != nil {
		_ = s.httpServer.Shutdown(ctx)
	}

	if s.otelProvider != nil {
		_ = s.otelProvider.Shutdown(ctx)
	}

	return s.Bundle.Close()
}

// SeedDemo creates a sample task so the API has data on first run.
func (s *Server) SeedDemo(ctx context.Context) {
	taskID := id.NewAggregateID()

	if err := s.Repo.Execute(
		ctx, taskID, aggregateType,
		Create(CreateTask{ID: taskID, Title: "Try the API!", Priority: PriorityHigh}),
	); err != nil {
		s.Logger.Warn("seed: create demo task", "error", err)

		return
	}

	s.Logger.Info(
		"seeded demo task",
		"id",
		taskID,
		"url",
		"http://localhost:8080/api/tasks/"+taskID.String(),
	)
}

// Run is the main entry point — wires, starts, waits for signal.
func Run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := DefaultConfig()
	if p := os.Getenv("DATABASE_PATH"); p != "" {
		cfg.DatabasePath = p
	}

	srv, err := NewServer(cfg, logger)
	if err != nil {
		return err
	}
	defer func() { _ = srv.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		return err
	}

	if err := srv.StartHTTP(":8080"); err != nil {
		return err
	}

	// Give projection a moment to replay, then seed
	time.Sleep(100 * time.Millisecond)
	srv.SeedDemo(ctx)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("signal received, shutting down", "signal", sig)

	return nil
}

// Ensure event package is used (for WithCommandCausality etc. in handlers).
var _ = event.Version(0)
