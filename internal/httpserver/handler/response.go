package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
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
) {
	writeJSON(logger, w, r, statusCode, dto.ErrorResponse{
		Error:   message,
		MaxSize: model.MaxAvatarFileSizeBytes,
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
