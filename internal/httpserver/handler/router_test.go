package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestNewRouter_UploadAvatarRoute проверяет регистрацию маршрута загрузки аватарки.
func TestNewRouter_UploadAvatarRoute(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	router := NewRouter(uploader, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, uploader.inputs, 1)
}

// TestNewRouter_UploadAvatarRoute_StripsTrailingSlash проверяет нормализацию завершающего слеша в маршруте загрузки
// аватарки.
func TestNewRouter_UploadAvatarRoute_StripsTrailingSlash(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	router := NewRouter(uploader, discardLogger())
	request := newUploadAvatarRequestToPath(
		t,
		avatarRoutePath+"/",
		testUserID.String(),
		"avatar.png",
		pngContent(),
	)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, uploader.inputs, 1)
}

// TestNewRouter_ReturnsMethodNotAllowed проверяет ошибку неподдержанного HTTP-метода.
func TestNewRouter_ReturnsMethodNotAllowed(t *testing.T) {
	// Arrange
	router := NewRouter(&avatarUploaderFake{}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, avatarRoutePath, nil)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
}

// TestNewRouter_UploadAvatarRoute_ReturnsUnsupportedMediaType проверяет ошибку неподдерживаемого content-type.
func TestNewRouter_UploadAvatarRoute_ReturnsUnsupportedMediaType(t *testing.T) {
	// Arrange
	router := NewRouter(&avatarUploaderFake{}, discardLogger())
	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", testUserID.String())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
}

// TestNewRouter_LogsRequest проверяет логирование HTTP-запроса.
func TestNewRouter_LogsRequest(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logBuffer, nil))
	router := NewRouter(&avatarUploaderFake{}, logger)
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	request.Header.Set("X-Real-IP", "203.0.113.10")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, `msg="http request"`)
	assert.Contains(t, logOutput, "method=POST")
	assert.Contains(t, logOutput, "uri=/api/v1/avatars")
	assert.Contains(t, logOutput, "ip=203.0.113.10")
	assert.Contains(t, logOutput, "status=201")
	assert.Contains(t, logOutput, "size=")
}

// TestNewRouter_UploadAvatarRoute_ReadsGzipRequest проверяет, что middleware распаковывает gzip-тело multipart-запроса
// до вызова обработчика загрузки аватарки.
func TestNewRouter_UploadAvatarRoute_ReadsGzipRequest(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	router := NewRouter(uploader, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	gzipRequestBody(t, request)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, uploader.inputs, 1)
}

// TestNewRouter_UploadAvatarRoute_RejectsInvalidGzipRequest проверяет ошибку невалидного gzip-тела запроса.
func TestNewRouter_UploadAvatarRoute_RejectsInvalidGzipRequest(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	router := NewRouter(uploader, discardLogger())
	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, bytes.NewBufferString("not gzip"))
	request.Header.Set("Content-Type", "multipart/form-data")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("X-User-ID", testUserID.String())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, uploader.inputs)
	assertErrorResponse(t, response, "Invalid gzip request body")
}

// TestNewRouter_UploadAvatarRoute_WritesGzipResponse проверяет gzip-сжатие ответа при поддержке gzip клиентом.
func TestNewRouter_UploadAvatarRoute_WritesGzipResponse(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{
		output: usecase.UploadAvatarOutput{
			ID:     testAvatarID,
			UserID: testUserID,
			Status: model.AvatarStatusProcessing,
		},
	}
	router := NewRouter(uploader, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, "gzip", response.Header().Get("Content-Encoding"))
	assert.Contains(t, gunzipResponseBody(t, response), `"status":"processing"`)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

func gzipRequestBody(t *testing.T, request *http.Request) {
	t.Helper()

	var body bytes.Buffer
	gzipWriter := gzip.NewWriter(&body)
	_, err := io.Copy(gzipWriter, request.Body)
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	request.Body = io.NopCloser(&body)
	request.ContentLength = int64(body.Len())
	request.Header.Set("Content-Encoding", "gzip")
}

func gunzipResponseBody(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()

	gzipReader, err := gzip.NewReader(response.Body)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, gzipReader.Close())
	}()

	body, err := io.ReadAll(gzipReader)
	require.NoError(t, err)

	return string(body)
}
