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
	readinessCheckTimeout = 2 * time.Second
	healthStatusOK        = "ok"
	healthStatusError     = "error"
	healthStatusDegraded  = "degraded"
)

// HealthHandler обрабатывает HTTP-запросы проверки жизнеспособности и готовности сервиса.
type HealthHandler struct {
	readinessChecks map[string]func(context.Context) error
	logger          *slog.Logger
}

// readinessCheckResult содержит имя зависимости и результат проверки её готовности.
type readinessCheckResult struct {
	name string
	err  error
}

// NewHealthHandler создает HealthHandler.
func NewHealthHandler(readinessChecks map[string]func(context.Context) error, logger *slog.Logger) (*HealthHandler, error) {
	if readinessChecks == nil {
		return nil, errors.New("readiness checks are not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &HealthHandler{
		readinessChecks: readinessChecks,
		logger:          logger.With("component", "httpserver.health_handler"),
	}, nil
}

// getLive подтверждает, что процесс сервиса запущен и не требует перезапуска.
func (h *HealthHandler) getLive(w http.ResponseWriter, r *http.Request) {
	writeJSON(h.logger, w, r, http.StatusOK, dto.LivenessResponse{Status: healthStatusOK})
}

// getReady выполняет именованные проверки внешних зависимостей и возвращает готовность сервиса принимать трафик.
func (h *HealthHandler) getReady(w http.ResponseWriter, r *http.Request) {
	var checkErr error
	statusCode := http.StatusOK
	response := dto.ReadinessResponse{
		Status: healthStatusOK,
		Checks: make(map[string]string, len(h.readinessChecks)),
	}

	resultCh := make(chan readinessCheckResult, len(h.readinessChecks))
	for name, check := range h.readinessChecks {
		go func() {
			ctx, cancel := context.WithTimeout(r.Context(), readinessCheckTimeout)
			defer cancel()

			resultCh <- readinessCheckResult{name: name, err: check(ctx)}
		}()
	}

	for range h.readinessChecks {
		result := <-resultCh
		response.Checks[result.name] = readinessCheckStatus(result.err)
		if result.err != nil {
			checkErr = errors.Join(checkErr, result.err)
		}
	}

	if checkErr != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = healthStatusDegraded
		logError(h.logger, r, "readiness check failed", checkErr)
	}

	writeJSON(h.logger, w, r, statusCode, response)
}

func readinessCheckStatus(err error) string {
	if err != nil {
		return healthStatusError
	}
	return healthStatusOK
}
