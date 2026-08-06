package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/readiness"
)

const (
	healthStatusOK       = "ok"
	healthStatusError    = "error"
	healthStatusDegraded = "degraded"
)

// HealthHandler обрабатывает HTTP-запросы проверки жизнеспособности и готовности сервиса.
type HealthHandler struct {
	readinessChecks readiness.Checks
	logger          *slog.Logger
}

// NewHealthHandler создает HealthHandler.
func NewHealthHandler(readinessChecks readiness.Checks, logger *slog.Logger) (*HealthHandler, error) {
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
	statusCode := http.StatusOK
	response := dto.ReadinessResponse{
		Status: healthStatusOK,
		Checks: make(map[string]string, len(h.readinessChecks)),
	}

	results := readiness.Run(r.Context(), h.readinessChecks, readiness.DefaultTimeout)
	for name, err := range results {
		response.Checks[name] = readinessCheckStatus(err)
	}

	checkErr := results.Err()
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
