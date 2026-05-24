package main

import (
	"log/slog"
	"net/http"
	"time"

	httputil "github.com/larsartmann/httputil"
)

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
	return httputil.CORS(httputil.CORSConfig{
		AllowedOrigins:     []string{"*"},
		AllowAllOrigins:    true,
		AllowedMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:     []string{"Content-Type", "Authorization"},
		ExposedHeaders:     []string{},
		AllowCredentials:   false,
		MaxAge:             86400,
		OptionsPassthrough: false,
	})(next)
}
