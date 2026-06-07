//go:build ignore

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
	"github.com/larsartmann/go-cqrs-lite/pkg/gracefulshutdown"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// runServer starts an HTTP server with operational endpoints.
// This demonstrates how consumers can expose health, metrics, and graceful
// shutdown in production.
func runServer(
	cmdDisp *command.Dispatcher,
	qryDisp *query.Dispatcher,
	store event.EventSource,
	bus event.EventBus,
) {
	metricsCollector := middleware.NewMetricsCollector()

	mux := http.NewServeMux()

	// Health endpoints
	mux.Handle("/health", middleware.HealthCheckHandler("v2.1.0"))
	mux.Handle("/health/live", middleware.HealthCheckHandler("v2.1.0"))
	mux.Handle("/health/ready", middleware.HealthCheckHandler("v2.1.0", dbHealthCheck(store)))

	// Metrics endpoint
	mux.Handle("/metrics", middleware.MetricsHandler(metricsCollector))

	// Wrap all handlers with metrics middleware
	wrapped := middleware.MetricsMiddleware(metricsCollector)(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      wrapped,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println("[server] Starting HTTP server on :8080")
	fmt.Println("[server]  - Health:  http://localhost:8080/health")
	fmt.Println("[server]  - Metrics: http://localhost:8080/metrics")

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[server] ListenAndServe error: %v\n", err)
		}
	}()

	gracefulshutdown.Shutdown(gracefulshutdown.DefaultConfig(), func(_ context.Context) error {
		fmt.Println("[server] Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	})
}

func dbHealthCheck(store event.EventSource) middleware.HealthChecker {
	return func(_ context.Context) middleware.Check {
		status := middleware.HealthStatusPass
		if store == nil {
			status = middleware.HealthStatusFail
		}
		return middleware.Check{
			ComponentID:   "event-store",
			ComponentType: "datastore",
			Status:        status,
		}
	}
}
