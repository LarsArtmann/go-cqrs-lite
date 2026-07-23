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

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"
)

// Server is the composition root — all wired components live here.
type Server struct {
	Bundle       *stack.Bundle
	Repo         *decider.Repository[TaskState]
	CmdDisp      *command.Dispatcher
	ReadModel    *kv.TypedStore[TaskView, TaskID]
	Mat          *stack.Materialize[TaskView, TaskID]
	ProjHost     *projectionhost.Host
	Logger       *slog.Logger
	otelProvider *cqrsotel.Provider
	signer       signing.SignerVerifier
	idemStore    idempotency.Store
	httpServer   *http.Server
	sseBroker    *cqrshttp.SSEBroker
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
		sqlite.WithPragmas(sqlopt.WithOptimizations()),
	)
	if err != nil {
		return nil, fmt.Errorf("setup: sqlite bundle: %w", err)
	}

	// SQLite's connection pool can create separate databases for :memory:.
	// Restricting to a single connection ensures schema visibility.
	if db, ok := bundle.Database().(*sql.DB); ok {
		db.SetMaxOpenConns(1)
	}

	// ── Repository with snapshot strategy (every 10 events) ──────────
	// Snapshots accelerate aggregate loading by folding from the last
	// snapshot instead of the full event history.
	snapStrategy, err := snapshot.EveryNEvents(10)
	if err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("setup: snapshot strategy: %w", err)
	}

	repo, err := stack.Repository(
		bundle, TaskDecider,
		decider.WithSnapshotStore[TaskState](bundle.SnapshotStore),
		decider.WithSnapshotStrategy[TaskState](snapStrategy),
		decider.WithCodec[TaskState](bundle.DefaultCodec()),
	)
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

	// ── ProjectionHost: managed projection runner with DLQ + crash-restart ──
	// Replaces the raw CatchUpSubscriber loop. The host reads from the
	// journal (replay), then tails live events via the subscriber. Poison
	// messages are captured in the dead-letter store after 3 failures.
	dlq := projectionhost.NewMemoryDeadLetterStore()

	projHost, err := projectionhost.New(
		bundle.SeekableJournal,
		bundle.CheckpointStore,
		projectionhost.WithBatchSize(100),
		projectionhost.WithDeadLetterStore(dlq, 3),
		projectionhost.WithMaxRestarts(-1),
		projectionhost.WithLogger(logger),
		projectionhost.WithSubscriber(bundle.Subscriber),
	)
	if err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("setup: projection host: %w", err)
	}

	if err := projHost.Register(mat); err != nil {
		_ = bundle.Close()

		return nil, fmt.Errorf("setup: register projection: %w", err)
	}

	srv := &Server{
		Bundle:    bundle,
		Repo:      repo,
		CmdDisp:   command.NewDispatcher(),
		ReadModel: rmStore,
		Mat:       mat,
		ProjHost:  projHost,
		Logger:    logger,
	}

	if err := setupFeatures(srv); err != nil {
		_ = srv.Bundle.Close()

		return nil, fmt.Errorf("setup: features: %w", err)
	}

	registerHandlers(srv)

	// Deriver: auto-assign new tasks to default team lead (event→command reaction)
	if err := projHost.Register(newDeriverProjection(srv.CmdDisp, logger)); err != nil {
		_ = srv.Bundle.Close()

		return nil, fmt.Errorf("setup: register deriver: %w", err)
	}

	// ── SSE broker: real-time event streaming over HTTP ──────────────
	// Clients connect to GET /events to receive a live stream of domain
	// events as Server-Sent Events.
	if bus, ok := bundle.Publisher.(event.Bus); ok {
		broker, brokerErr := cqrshttp.NewSSEBroker(bus)
		if brokerErr != nil {
			return nil, fmt.Errorf("setup: SSE broker: %w", brokerErr)
		}

		srv.sseBroker = broker
	}

	return srv, nil
}

// Start launches the ProjectionHost (replay + live tailing with DLQ).
// Call StartHTTP separately for the HTTP API (not needed in tests).
func (s *Server) Start(ctx context.Context) error {
	go func() {
		if err := s.ProjHost.Start(ctx); err != nil {
			s.Logger.Error("projection host", "error", err)
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

		if err := s.httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			s.Logger.Error("HTTP server", "error", err)
		}
	}()

	return nil
}

// Stop gracefully drains all components.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.ProjHost != nil {
		_ = s.ProjHost.Stop()
	}

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
		ctx, taskID, streamType,
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
