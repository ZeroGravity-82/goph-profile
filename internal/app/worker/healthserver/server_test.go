package healthserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/readiness"
)

// TestServer_getLive проверяет успешный ответ ручки жизнеспособности без проверки внешних зависимостей.
func TestServer_getLive(t *testing.T) {
	// Arrange
	checkCalled := false
	server := mustServer(t, "127.0.0.1:0", readiness.Checks{
		"dependency": func(_ context.Context) error {
			checkCalled = true
			return errors.New("dependency error")
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	response := httptest.NewRecorder()

	// Act
	server.handler().ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	assert.False(t, checkCalled)
}

// TestServer_getReady проверяет успешный ответ ручки готовности при доступности всех зависимостей.
func TestServer_getReady(t *testing.T) {
	// Arrange
	server := mustServer(t, "127.0.0.1:0", okReadinessChecks())
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()

	// Act
	server.handler().ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.JSONEq(t, `{
		"status":"ok",
		"checks":{"postgres":"ok","s3":"ok","rabbitmq":"ok"}
	}`, response.Body.String())
}

// TestServer_getReady_ReturnsServiceUnavailable проверяет ответ ручки готовности при недоступной зависимости.
func TestServer_getReady_ReturnsServiceUnavailable(t *testing.T) {
	// Arrange
	server := mustServer(t, "127.0.0.1:0", readiness.Checks{
		"postgres": func(_ context.Context) error { return nil },
		"s3":       func(_ context.Context) error { return errors.New("s3 error") },
		"rabbitmq": func(_ context.Context) error { return nil },
	})
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	response := httptest.NewRecorder()

	// Act
	server.handler().ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.JSONEq(t, `{
		"status":"degraded",
		"checks":{"postgres":"ok","s3":"error","rabbitmq":"ok"}
	}`, response.Body.String())
}

// TestServer_Run_ReturnsListenError проверяет ошибку запуска сервера проверок состояния.
func TestServer_Run_ReturnsListenError(t *testing.T) {
	// Arrange
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	server := mustServer(t, listener.Addr().String(), okReadinessChecks())

	// Act
	err = server.Run(context.Background())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run worker health server")
}

// TestServer_Run_ShutsDownOnContextCancel проверяет graceful shutdown сервера проверок состояния.
func TestServer_Run_ShutsDownOnContextCancel(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	server := mustServer(t, "127.0.0.1:0", okReadinessChecks())
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- server.Run(ctx)
	}()

	// Act
	cancel()

	// Assert
	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("worker health server did not stop after context cancellation")
	}
}

func mustServer(t *testing.T, addr string, checks readiness.Checks) *Server {
	t.Helper()

	server, err := New(addr, checks, nil)
	require.NoError(t, err)
	return server
}

func okReadinessChecks() readiness.Checks {
	return readiness.Checks{
		"postgres": func(_ context.Context) error { return nil },
		"s3":       func(_ context.Context) error { return nil },
		"rabbitmq": func(_ context.Context) error { return nil },
	}
}
