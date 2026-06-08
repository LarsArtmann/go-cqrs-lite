package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
	"github.com/larsartmann/go-cqrs-lite/pkg/gracefulshutdown"
)

const serverVersion = "v2.2.0"

func runServer(store event.EventSource) {
	metricsCollector := middleware.NewMetricsCollector()

	mux := http.NewServeMux()

	mux.Handle("/health", middleware.HealthCheckHandler(
		serverVersion,
		dbHealthCheck(store),
	))
	mux.Handle("/health/live", middleware.HealthCheckHandler(serverVersion))
	mux.Handle("/health/ready", middleware.HealthCheckHandler(
		serverVersion,
		dbHealthCheck(store),
	))

	mux.Handle("/metrics", middleware.MetricsHandler(metricsCollector))

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
