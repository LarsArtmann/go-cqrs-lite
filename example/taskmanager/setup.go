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
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4" // register "sqlite" driver
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-idempotency"
	gomust "github.com/larsartmann/go-must"
	otel "go.opentelemetry.io/otel"
)

const (
	snapshotInterval     = 10
	projectionBatchSize  = 100
	deadLetterMaxRetries = 3
	httpReadHeaderSecs   = 5
	shutdownTimeoutSecs  = 30
	projectionSettleMs   = 100
	primaryEngine        = "primary"
	sseReplayCapacity    = 256
	sseHeartbeatSecs     = 30
)

// Server holds the System and HTTP lifecycle. Infrastructure (event store,
// snapshot store, projection host, bus, dispatchers) is owned by the System.
type Server struct {
	Sys          *system.System
	CmdDisp      *command.Dispatcher
	MetaEngine   *metaengine.Store
	TaskReader   *metaengine.TypedReader[TaskView]
	ProjHost     *projectionhost.Host
	Logger       *slog.Logger
	otelProvider *cqrsotel.Provider
	signer       signing.SignerVerifier
	httpServer   *http.Server
	taskWatcher  *metaengine.Watcher[TaskView]
}

// Config configures the Server.
type Config struct {
	DatabasePath string
	HTTPAddr     string
}

func DefaultConfig() Config {
	return Config{DatabasePath: ":memory:", HTTPAddr: ":8080"}
}

// NewServer wires all components via system.New and returns a ready-to-start Server.
func NewServer(cfg Config, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// ── OTel setup (before system.New so middleware can be in DomainConfig) ──
	provider, err := cqrsotel.Setup(
		cqrsotel.WithService("taskmanager", "1.0.0", "dev"),
	)
	if err != nil {
		return nil, fmt.Errorf("setup: otel: %w", err)
	}

	tracer := otel.GetTracerProvider().Tracer("taskmanager")
	meter := otel.GetMeterProvider().Meter("taskmanager")

	otelBundle, err := middleware.NewOTelBundle(tracer, meter)
	if err != nil {
		return nil, fmt.Errorf("setup: otel bundle: %w", err)
	}

	// ── Build domain config ──────────────────────────────────────────────
	projections, typeDecoder := buildProjections()

	snapStrategy, err := snapshot.EveryNEvents(snapshotInterval)
	if err != nil {
		return nil, fmt.Errorf("setup: snapshot strategy: %w", err)
	}

	dlq := projectionhost.NewMemoryDeadLetterStore()

	const idempotencyTTL = 10 * time.Minute

	commandMW := []command.Middleware{ //nolint:prealloc // small fixed set + append
		middleware.CommandRecovery(),
		middleware.CommandLogging(logger),
		middleware.CommandRetry(middleware.DefaultRetryConfig()),
		middleware.CommandIdempotency(idempotency.NewMemoryStore(0), idempotencyTTL, nil),
	}
	commandMW = append(commandMW, otelBundle.Command()...)

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			gomust.Check(system.RegisterDecider(sys, string(streamType), TaskDecider,
				system.WithSnapshotStrategy(snapStrategy)))
			registerCommands(sys)
		},
		Projections:           projections,
		ProjectionTypeDecoder: typeDecoder,
		Middleware:            commandMW,
		ProjectionHostOptions: []projectionhost.HostOption{
			projectionhost.WithBatchSize(projectionBatchSize),
			projectionhost.WithDeadLetterStore(dlq, deadLetterMaxRetries),
			projectionhost.WithMaxRestarts(-1),
			projectionhost.WithLogger(logger),
		},
	}

	// ── Build deployment config ──────────────────────────────────────────
	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			primaryEngine: {
				Driver:  "sqlite",
				DSN:     cfg.DatabasePath,
				Pragmas: []string{"journal_mode=wal"},
			},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: primaryEngine},
			{Role: system.RoleProjections, Engine: primaryEngine},
		},
	}

	// ── Create the System ────────────────────────────────────────────────
	ctx := context.Background()

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		return nil, fmt.Errorf("setup: system.New: %w", err)
	}

	srv := &Server{
		Sys:          sys,
		CmdDisp:      sys.CommandDispatcher(),
		MetaEngine:   sys.MetaEngine(),
		ProjHost:     sys.ProjectionHost(),
		Logger:       logger,
		otelProvider: provider,
	}

	if sys.MetaEngine() != nil {
		srv.TaskReader = metaengine.NewReader[TaskView](sys.MetaEngine(), "task_views")
	}

	// ── Event bus signing (HMAC-SHA256 tamper detection) ─────────────────
	srv.signer = newDemoSigner()

	if err := sys.Bus().UsePublish(signing.SignMiddleware(srv.signer)); err != nil {
		return nil, fmt.Errorf("setup: sign middleware: %w", err)
	}

	if err := sys.Bus().Use(signing.VerifyMiddleware(srv.signer)); err != nil {
		return nil, fmt.Errorf("setup: verify middleware: %w", err)
	}

	// ── Deriver: auto-assign new tasks (event→command reaction) ──────────
	if err := sys.ProjectionHost().Register(
		newDeriverProjection(sys.CommandDispatcher(), logger),
	); err != nil {
		return nil, fmt.Errorf("setup: register deriver: %w", err)
	}

	// ── SSE: live read-model updates via metaengine watcher (ADR-0127) ──
	// Streams TaskView changes over go-sse (metaengine.ServeSSE), with
	// Last-Event-ID reconnection via the replay journal. Replaces the
	// deprecated transport/http.SSEBroker raw-event stream.
	if sys.MetaEngine() != nil {
		srv.taskWatcher = metaengine.NewWatcher[TaskView](sys.MetaEngine(), "task_views")
		srv.taskWatcher.WithReplay(sseReplayCapacity)
	}

	return srv, nil
}

// Start launches the ProjectionHost in a background goroutine.
func (s *Server) Start(ctx context.Context) {
	go func() {
		if err := s.ProjHost.Start(ctx); err != nil {
			s.Logger.Error("projection host", "error", err)
		}
	}()
}

// StartHTTP launches the HTTP API server in a background goroutine.
func (s *Server) StartHTTP(addr string) {
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: httpReadHeaderSecs * time.Second,
	}

	go func() {
		s.Logger.Info("HTTP server starting", "addr", s.httpServer.Addr)

		if err := s.httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			s.Logger.Error("HTTP server", "error", err)
		}
	}()
}

// Stop gracefully drains all components.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeoutSecs*time.Second)
	defer cancel()

	if s.ProjHost != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = s.ProjHost.Stop()
	}

	if s.httpServer != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = s.httpServer.Shutdown(ctx)
	}

	if s.otelProvider != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = s.otelProvider.Shutdown(ctx)
	}

	if s.taskWatcher != nil {
		s.taskWatcher.Close()
	}

	return s.Sys.Close()
}

// SeedDemo creates a sample task so the API has data on first run.
func (s *Server) SeedDemo(ctx context.Context) {
	taskID := id.NewStreamID()

	if err := s.CmdDisp.Dispatch(ctx, CreateTaskCmd{
		BasicCommand: gomust.Must(command.New(cmdCreateTask, taskID)),
		Title:        "Try the API!",
		Priority:     PriorityHigh,
	}); err != nil {
		s.Logger.Warn("seed: create demo task", "error", err)

		return
	}

	s.Logger.Info(
		"seeded demo task",
		"id", taskID,
		"url", "http://localhost:8080/api/tasks/"+taskID.String(),
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

	srv.Start(ctx)
	srv.StartHTTP(":8080")

	time.Sleep(projectionSettleMs * time.Millisecond)
	srv.SeedDemo(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("signal received, shutting down", "signal", sig)

	return nil
}
