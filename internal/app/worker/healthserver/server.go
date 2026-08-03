// Package healthserver предоставляет HTTP-сервер проверок жизнеспособности и готовности фонового процесса.
package healthserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

const (
	readHeaderTimeout     = 5 * time.Second
	readinessCheckTimeout = 2 * time.Second
	shutdownTimeout       = 5 * time.Second
	statusOK              = "ok"
	statusError           = "error"
	statusDegraded        = "degraded"
)

// ReadinessCheck проверяет готовность зависимости воркера.
type ReadinessCheck func(ctx context.Context) error

// ReadinessChecks содержит именованные проверки готовности зависимостей воркера.
type ReadinessChecks map[string]ReadinessCheck

// Server обслуживает проверки жизнеспособности и готовности воркера.
type Server struct {
	addr            string
	readinessChecks ReadinessChecks
	logger          *slog.Logger
}

// livenessResponse описывает JSON-ответ ручки жизнеспособности воркера.
type livenessResponse struct {
	Status string `json:"status"`
}

// readinessResponse описывает JSON-ответ ручки готовности воркера.
type readinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// New создает сервер проверок состояния воркера.
func New(addr string, readinessChecks ReadinessChecks, logger *slog.Logger) (*Server, error) {
	if addr == "" {
		return nil, errors.New("health server address is not provided")
	}
	if len(readinessChecks) == 0 {
		return nil, errors.New("readiness checks are not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &Server{
		addr:            addr,
		readinessChecks: readinessChecks,
		logger:          logger.With("component", "worker.healthserver"),
	}, nil
}

// Run запускает HTTP-сервер проверок состояния и блокируется до отмены контекста или ошибки HTTP-сервера.
func (s *Server) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.addr,
		Handler:           s.handler(),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	errCh := make(chan error, 1)
	go func() {
		s.logger.InfoContext(ctx, "starting worker health server", slog.String("addr", s.addr))
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := server.Shutdown(shutdownCtx)
		if err == nil {
			s.logger.InfoContext(shutdownCtx, "worker health server stopped with graceful shutdown")
			return nil
		}
		return fmt.Errorf("failed to stop worker health server: %w", err)
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			s.logger.InfoContext(ctx, "http server closed")
			return nil
		}
		return fmt.Errorf("failed to run worker health server: %w", err)
	}
}

// handler собирает маршруты проверок жизнеспособности и готовности воркера.
func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /live", s.getLive)
	mux.HandleFunc("GET /ready", s.getReady)
	return mux
}

// getLive подтверждает, что процесс воркера запущен и не требует перезапуска.
func (s *Server) getLive(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, r, http.StatusOK, livenessResponse{Status: statusOK})
}

// writeJSON записывает JSON-ответ сервера проверок состояния.
func (s *Server) writeJSON(w http.ResponseWriter, r *http.Request, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		s.logger.ErrorContext(r.Context(), "failed to write worker health response", slog.Any("err", err))
	}
}

// getReady выполняет проверки зависимостей и возвращает готовность воркера обрабатывать сообщения.
func (s *Server) getReady(w http.ResponseWriter, r *http.Request) {
	var checkErr error
	statusCode := http.StatusOK
	response := readinessResponse{
		Status: statusOK,
		Checks: make(map[string]string, len(s.readinessChecks)),
	}

	for name, check := range s.readinessChecks {
		ctx, cancel := context.WithTimeout(r.Context(), readinessCheckTimeout)
		err := check(ctx)
		cancel()

		response.Checks[name] = readinessCheckStatus(err)
		if err != nil {
			checkErr = errors.Join(checkErr, err)
		}
	}

	if checkErr != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = statusDegraded
		s.logger.ErrorContext(r.Context(), "worker readiness check failed", slog.Any("err", checkErr))
	}

	s.writeJSON(w, r, statusCode, response)
}

// readinessCheckStatus преобразует результат проверки зависимости в статус ответа.
func readinessCheckStatus(err error) string {
	if err != nil {
		return statusError
	}
	return statusOK
}
