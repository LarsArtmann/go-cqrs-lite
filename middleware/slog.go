package middleware

import (
	"log/slog"
)

// SlogAdapter adapts a *slog.Logger to the Logger interface used by middleware.
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	mw := middleware.CommandLogging(middleware.SlogAdapter(logger))
func SlogAdapter(logger *slog.Logger) Logger {
	return &slogLogger{logger: logger}
}

type slogLogger struct {
	logger *slog.Logger
}

func (s *slogLogger) Info(msg string, keyvals ...any) {
	s.logger.Info(msg, keyvals...)
}

func (s *slogLogger) Error(msg string, keyvals ...any) {
	s.logger.Error(msg, keyvals...)
}
