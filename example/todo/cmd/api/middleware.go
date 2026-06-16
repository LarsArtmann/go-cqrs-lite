package main

import (
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"
)

const corsMaxAgeSeconds = 86400

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)
			logger.Info(
				"HTTP",
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
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(corsMaxAgeSeconds))

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func chainMiddleware(
	final http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) http.Handler {
	for _, v := range slices.Backward(middlewares) {
		final = v(final)
	}

	return final
}
