package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/go-chi/chi/v5/middleware"
)

func withLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = logging.NopLogger()
	}

	return middleware.RequestLogger(slogLogFormatter{logger: logger})
}

type slogLogFormatter struct {
	logger *slog.Logger
}

func (f slogLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return slogLogEntry{
		logger:  f.logger,
		request: r,
	}
}

type slogLogEntry struct {
	logger  *slog.Logger
	request *http.Request
}

func (e slogLogEntry) Write(
	status int,
	bytes int,
	_ http.Header,
	elapsed time.Duration,
	_ interface{},
) {
	if status == 0 {
		status = http.StatusOK
	}

	e.logger.InfoContext(
		e.request.Context(),
		"http request",
		slog.String("method", e.request.Method),
		slog.String("uri", e.request.RequestURI),
		slog.String("ip", e.request.RemoteAddr),
		slog.Duration("duration", elapsed),
		slog.Int("status", status),
		slog.Int("size", bytes),
	)
}

func (e slogLogEntry) Panic(v interface{}, stack []byte) {
	e.logger.ErrorContext(
		e.request.Context(),
		"http panic",
		slog.Any("panic", v),
		slog.String("stack", string(stack)),
	)
}
