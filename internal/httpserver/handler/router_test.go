package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestNewRouter_UploadAvatarRoute_WithNilLogger проверяет работу роутера без переданного логгера.
func TestNewRouter_UploadAvatarRoute_WithNilLogger(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	router := NewRouter(uploader, nil)
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
