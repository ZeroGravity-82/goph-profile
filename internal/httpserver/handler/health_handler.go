package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

const (
	healthCheckTimeout   = 2 * time.Second
	healthStatusOK       = "ok"
	healthStatusError    = "error"
	healthStatusDegraded = "degraded"
)

// HealthHandler обрабатывает HTTP-запрос проверки состояния сервиса и его внешних зависимостей.
type HealthHandler struct {
	checks map[string]func(context.Context) error
	logger *slog.Logger
}

// NewHealthHandler создает HealthHandler.
func NewHealthHandler(checks map[string]func(context.Context) error, logger *slog.Logger) (*HealthHandler, error) {
	if checks == nil {
		return nil, errors.New("health checks are not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &HealthHandler{checks: checks, logger: logger}, nil
}

// getHealth выполняет именованные проверки внешних зависимостей и возвращает общий статус сервиса.
func (h *HealthHandler) getHealth(w http.ResponseWriter, r *http.Request) {
	var checkErr error
	statusCode := http.StatusOK
	response := dto.HealthResponse{
		Status: healthStatusOK,
		Checks: make(map[string]string, len(h.checks)),
	}

	for name, check := range h.checks {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		err := check(ctx)
		cancel()

		response.Checks[name] = healthCheckStatus(err)
		if err != nil {
			checkErr = errors.Join(checkErr, err)
		}
	}

	if checkErr != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = healthStatusDegraded
		logError(h.logger, r, "health check failed", checkErr)
	}

	writeJSON(h.logger, w, r, statusCode, response)
}

func healthCheckStatus(err error) string {
	if err != nil {
		return healthStatusError
	}
	return healthStatusOK
}
