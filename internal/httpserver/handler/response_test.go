package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
)

type errorResponseWriter struct {
	header     http.Header
	statusCode int
}

func (w *errorResponseWriter) Header() http.Header {
	return w.header
}

func (w *errorResponseWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

func (w *errorResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// TestWriteError проверяет JSON-ответ с ошибкой и деталями.
func TestWriteError(t *testing.T) {
	// Arrange
	request := httptest.NewRequest(http.MethodGet, avatarRoutePath, nil)
	response := httptest.NewRecorder()

	// Act
	writeError(discardLogger(), response, request, http.StatusBadRequest, "Invalid request", "missing id")

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	var body dto.ErrorResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "Invalid request", body.Error)
	assert.Equal(t, "missing id", body.Details)
}

// TestWriteErrorWithMaxSize проверяет JSON-ответ с ошибкой и максимальным допустимым размером.
func TestWriteErrorWithMaxSize(t *testing.T) {
	// Arrange
	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, nil)
	response := httptest.NewRecorder()

	// Act
	writeErrorWithMaxSize(
		discardLogger(),
		response,
		request,
		http.StatusRequestEntityTooLarge,
		"File too large",
		1024,
	)

	// Assert
	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))

	var body dto.ErrorResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "File too large", body.Error)
	assert.Equal(t, int64(1024), body.MaxSize)
}

// TestWriteJSON_LogsWriteError проверяет логирование ошибки записи JSON-ответа.
func TestWriteJSON_LogsWriteError(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	logger := newTextLogger(logBuffer)
	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, nil)
	response := &errorResponseWriter{header: http.Header{}}

	// Act
	writeJSON(logger, response, request, http.StatusCreated, dto.UploadAvatarResponse{ID: "avatar-id"})

	// Assert
	assert.Equal(t, "application/json", response.header.Get("Content-Type"))
	assert.Equal(t, http.StatusCreated, response.statusCode)
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "failed to write HTTP response")
	assert.Contains(t, logOutput, "write error")
	assert.Contains(t, logOutput, "status=201")
}
