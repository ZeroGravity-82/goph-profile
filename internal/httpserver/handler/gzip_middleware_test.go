package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithGzip_ReadsGzipRequest проверяет, что middleware распаковывает gzip-тело запроса для следующего handler.
func TestWithGzip_ReadsGzipRequest(t *testing.T) {
	// Arrange
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "request body", string(body))
		assert.Empty(t, r.Header.Get("Content-Encoding"))

		w.WriteHeader(http.StatusNoContent)
	})
	handler := withGzip(discardLogger())(next)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", gzipContent(t, []byte("request body")))
	request.Header.Set("Content-Encoding", "gzip")
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusNoContent, response.Code)
}

// TestWithGzip_RejectsInvalidGzipRequest проверяет ошибку невалидного gzip-тела запроса.
func TestWithGzip_RejectsInvalidGzipRequest(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := withGzip(newTextLogger(logBuffer))(next)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", bytes.NewBufferString("not gzip"))
	request.Header.Set("Content-Encoding", "gzip")
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assertErrorResponse(t, response, "Invalid gzip request body")
	assert.Contains(t, logBuffer.String(), "failed to create gzip request reader")
}

// TestWithGzip_WritesGzipResponse проверяет gzip-сжатие JSON-ответа при поддержке gzip клиентом.
func TestWithGzip_WritesGzipResponse(t *testing.T) {
	// Arrange
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":"ok"}`))
		require.NoError(t, err)
	})
	handler := withGzip(discardLogger())(next)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/avatars", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, "gzip", response.Header().Get("Content-Encoding"))
	assert.JSONEq(t, `{"status":"ok"}`, gunzipResponseBody(t, response))
}

// TestHasGzipEncoding проверяет поиск gzip среди значений Content-Encoding.
func TestHasGzipEncoding(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "empty", header: "", want: false},
		{name: "gzip", header: "gzip", want: true},
		{name: "case insensitive", header: "GZip", want: true},
		{name: "list", header: "br, gzip", want: true},
		{name: "unsupported", header: "br", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := hasGzipEncoding(tt.header)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}

func gzipContent(t *testing.T, content []byte) *bytes.Buffer {
	t.Helper()

	body := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(body)
	_, err := gzipWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	return body
}
