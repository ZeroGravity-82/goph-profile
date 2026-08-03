package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
)

// TestHealthHandler_getLive проверяет успешный ответ ручки жизнеспособности без проверки внешних зависимостей.
func TestHealthHandler_getLive(t *testing.T) {
	// Arrange
	checkCalled := false
	handler := mustHealthHandler(t, map[string]func(context.Context) error{
		"dependency": func(_ context.Context) error {
			checkCalled = true
			return errors.New("dependency error")
		},
	}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getLive(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	assert.False(t, checkCalled)
}

// TestHealthHandler_getReady проверяет успешный ответ ручки готовности при доступности всех внешних зависимостей.
func TestHealthHandler_getReady(t *testing.T) {
	// Arrange
	handler := mustHealthHandler(t, okReadinessChecks(), discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getReady(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assertReadinessResponse(t, response, dto.ReadinessResponse{
		Status: healthStatusOK,
		Checks: map[string]string{
			"postgres": healthStatusOK,
			"s3":       healthStatusOK,
			"rabbitmq": healthStatusOK,
		},
	})
}

// TestHealthHandler_getReady_RunsChecksConcurrently проверяет параллельный запуск проверок готовности зависимостей.
func TestHealthHandler_getReady_RunsChecksConcurrently(t *testing.T) {
	// Arrange
	var startedChecks atomic.Int32
	allChecksStarted := make(chan struct{})
	waitForAllChecks := func(ctx context.Context) error {
		if startedChecks.Add(1) == 2 {
			close(allChecksStarted)
		}
		select {
		case <-allChecksStarted:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	handler := mustHealthHandler(t, map[string]func(context.Context) error{
		"postgres": waitForAllChecks,
		"s3":       waitForAllChecks,
	}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getReady(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, int32(2), startedChecks.Load())
}

func assertReadinessResponse(t *testing.T, response *httptest.ResponseRecorder, want dto.ReadinessResponse) {
	t.Helper()

	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	var body dto.ReadinessResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, want, body)
}

// TestHealthHandler_getReady_ReturnsServiceUnavailable проверяет неготовность сервиса при недоступной зависимости.
func TestHealthHandler_getReady_ReturnsServiceUnavailable(t *testing.T) {
	// Arrange
	handler := mustHealthHandler(t, map[string]func(context.Context) error{
		"postgres": okReadinessCheck,
		"s3":       readinessCheckError(errors.New("s3 error")),
		"rabbitmq": okReadinessCheck,
	}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getReady(response, request)

	// Assert
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	assertReadinessResponse(t, response, dto.ReadinessResponse{
		Status: healthStatusDegraded,
		Checks: map[string]string{
			"postgres": healthStatusOK,
			"s3":       healthStatusError,
			"rabbitmq": healthStatusOK,
		},
	})
}
