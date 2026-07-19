package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

func writeError(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	message string,
	details string,
) {
	writeJSON(logger, w, r, statusCode, dto.ErrorResponse{
		Error:   message,
		Details: details,
	})
}

func writeErrorWithMaxSize(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	message string,
	maxSize int64,
) {
	writeJSON(logger, w, r, statusCode, dto.ErrorResponse{
		Error:   message,
		MaxSize: maxSize,
	})
}

func writeJSON(logger *slog.Logger, w http.ResponseWriter, r *http.Request, statusCode int, response any) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(
			r.Context(),
			"failed to write HTTP response",
			slog.Any("error", err),
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.Int("status", statusCode),
		)
	}
}

func logError(logger *slog.Logger, r *http.Request, message string, err error) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	logger.ErrorContext(
		r.Context(),
		message,
		slog.Any("error", err),
		slog.String("method", r.Method),
		slog.String("uri", r.RequestURI),
	)
}
