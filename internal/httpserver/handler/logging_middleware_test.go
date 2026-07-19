package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithLogging_LogsResponse проверяет, что middleware логирует параметры HTTP-ответа.
func TestWithLogging_LogsResponse(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("created"))
		require.NoError(t, err)
	})
	handler := withLogging(newTextLogger(logBuffer))(next)
	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, nil)
	request.RemoteAddr = "203.0.113.10"
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, `msg="http request"`)
	assert.Contains(t, logOutput, "method=POST")
	assert.Contains(t, logOutput, "uri=/api/v1/avatars")
	assert.Contains(t, logOutput, "ip=203.0.113.10")
	assert.Contains(t, logOutput, "status=201")
	assert.Contains(t, logOutput, "size=7")
}

// TestWithLogging_LogsDefaultStatus проверяет логирование 200 OK, если хендлер не записал ответ.
func TestWithLogging_LogsDefaultStatus(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	handler := withLogging(newTextLogger(logBuffer))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	request := httptest.NewRequest(http.MethodGet, avatarRoutePath, nil)
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "status=200")
	assert.Contains(t, logOutput, "size=0")
}

// TestSlogLogEntry_Panic проверяет логирование паники через LogEntry от chi.
func TestSlogLogEntry_Panic(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	request := httptest.NewRequest(http.MethodGet, avatarRoutePath, nil)
	entry := slogLogEntry{
		logger:  newTextLogger(logBuffer),
		request: request,
	}

	// Act
	entry.Panic("panic value", []byte("stack trace"))

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, `msg="http panic"`)
	assert.Contains(t, logOutput, `panic="panic value"`)
	assert.Contains(t, logOutput, `stack="stack trace"`)
}
