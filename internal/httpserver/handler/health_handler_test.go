package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
)

// TestHealthHandler_getHealth проверяет успешную проверку состояния сервиса.
func TestHealthHandler_getHealth(t *testing.T) {
	// Arrange
	handler := mustHealthHandler(t, okHealthChecks(), discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getHealth(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assertHealthResponse(t, response, dto.HealthResponse{
		Status: healthStatusOK,
		Checks: map[string]string{
			"postgres": healthStatusOK,
			"s3":       healthStatusOK,
			"rabbitmq": healthStatusOK,
		},
	})
}

func assertHealthResponse(t *testing.T, response *httptest.ResponseRecorder, want dto.HealthResponse) {
	t.Helper()

	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	var body dto.HealthResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, want, body)
}

// TestHealthHandler_getHealth_ReturnsServiceUnavailable проверяет ошибку при недоступной зависимости.
func TestHealthHandler_getHealth_ReturnsServiceUnavailable(t *testing.T) {
	// Arrange
	handler := mustHealthHandler(t, map[string]func(context.Context) error{
		"postgres": okHealthCheck,
		"s3":       healthCheckError(errors.New("s3 error")),
		"rabbitmq": okHealthCheck,
	}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getHealth(response, request)

	// Assert
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	assertHealthResponse(t, response, dto.HealthResponse{
		Status: healthStatusDegraded,
		Checks: map[string]string{
			"postgres": healthStatusOK,
			"s3":       healthStatusError,
			"rabbitmq": healthStatusOK,
		},
	})
}
