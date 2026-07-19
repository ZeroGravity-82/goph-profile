package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

var (
	testUserID   = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001")
	testAvatarID = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")
)

// TestAvatarHandler_uploadAvatar проверяет успешную загрузку аватарки.
func TestAvatarHandler_uploadAvatar(t *testing.T) {
	// Arrange
	createdAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	uploader := &avatarUploaderFake{
		output: usecase.UploadAvatarOutput{
			ID:        testAvatarID,
			UserID:    testUserID,
			Status:    model.AvatarStatusProcessing,
			CreatedAt: createdAt,
		},
	}
	handler := NewAvatarHandler(uploader, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	require.Len(t, uploader.inputs, 1)
	assert.Equal(t, testUserID, uploader.inputs[0].UserID)
	assert.Equal(t, "avatar.png", uploader.inputs[0].FileName)
	assert.Equal(t, model.MIMEPNG, uploader.inputs[0].MIMEType)
	assert.Equal(t, pngContent(), uploader.inputs[0].Content)

	var body uploadAvatarResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, testAvatarID.String(), body.ID)
	assert.Equal(t, testUserID.String(), body.UserID)
	assert.Equal(t, mustAvatarURL(t, testAvatarID.String()), body.URL)
	assert.Equal(t, string(model.AvatarStatusProcessing), body.Status)
	assert.Equal(t, createdAt, body.CreatedAt)
}

// TestAvatarHandler_uploadAvatar_RejectsInvalidUserID проверяет ошибку невалидного заголовка X-User-ID.
func TestAvatarHandler_uploadAvatar_RejectsInvalidUserID(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	handler := NewAvatarHandler(uploader, discardLogger())
	request := newUploadAvatarRequest(t, "not-a-uuid", "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, uploader.inputs)
	assertErrorResponse(t, response, "Invalid X-User-ID header")
}

// TestAvatarHandler_uploadAvatar_RejectsMissingFile проверяет ошибку отсутствующего файла.
func TestAvatarHandler_uploadAvatar_RejectsMissingFile(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	handler := NewAvatarHandler(uploader, discardLogger())
	request := newUploadAvatarRequestWithoutFile(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, uploader.inputs)
	assertErrorResponse(t, response, "Invalid file format")
}

// TestAvatarHandler_uploadAvatar_RejectsUnsupportedFormat проверяет ошибку неподдерживаемого формата файла.
func TestAvatarHandler_uploadAvatar_RejectsUnsupportedFormat(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	handler := NewAvatarHandler(uploader, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.txt", []byte("not an image"))
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, uploader.inputs)
	assertErrorResponse(t, response, "Invalid file format")
}

// TestAvatarHandler_uploadAvatar_RejectsTooLargeFile проверяет ошибку слишком большого файла.
func TestAvatarHandler_uploadAvatar_RejectsTooLargeFile(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{}
	handler := NewAvatarHandler(uploader, discardLogger())
	content := bytes.Repeat([]byte{0x89}, int(model.MaxAvatarFileSizeBytes)+1)
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", content)
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	assert.Empty(t, uploader.inputs)

	var body errorResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "File too large", body.Error)
	assert.Equal(t, model.MaxAvatarFileSizeBytes, body.MaxSize)
}

// TestAvatarHandler_uploadAvatar_ReturnsUserNotFound проверяет ошибку отсутствующего пользователя.
func TestAvatarHandler_uploadAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	uploader := &avatarUploaderFake{err: usecase.ErrUserNotFound}
	handler := NewAvatarHandler(uploader, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, uploader.inputs, 1)
	assertErrorResponse(t, response, "User not found")
}

// TestAvatarHandler_uploadAvatar_ReturnsInternalServerError проверяет внутреннюю ошибку загрузки аватарки.
func TestAvatarHandler_uploadAvatar_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	uploader := &avatarUploaderFake{err: errors.New("database error")}
	handler := NewAvatarHandler(uploader, newTextLogger(logBuffer))
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, uploader.inputs, 1)
	assertErrorResponse(t, response, "Internal server error")
	assert.Contains(t, logBuffer.String(), "database error")
}

// TestWriteJSON_LogsWriteError проверяет логирование ошибки записи JSON-ответа.
func TestWriteJSON_LogsWriteError(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	logger := newTextLogger(logBuffer)
	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, nil)
	response := &errorResponseWriter{header: http.Header{}}

	// Act
	writeJSON(logger, response, request, http.StatusCreated, uploadAvatarResponse{ID: testAvatarID.String()})

	// Assert
	assert.Equal(t, "application/json", response.header.Get("Content-Type"))
	assert.Equal(t, http.StatusCreated, response.statusCode)
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "failed to write HTTP response")
	assert.Contains(t, logOutput, "write error")
	assert.Contains(t, logOutput, "status=201")
}

type avatarUploaderFake struct {
	output usecase.UploadAvatarOutput
	err    error
	inputs []usecase.UploadAvatarInput
}

func (u *avatarUploaderFake) UploadAvatar(
	_ context.Context,
	input usecase.UploadAvatarInput,
) (usecase.UploadAvatarOutput, error) {
	u.inputs = append(u.inputs, input)
	return u.output, u.err
}

func newUploadAvatarRequest(
	t *testing.T,
	userID string,
	fileName string,
	content []byte,
) *http.Request {
	t.Helper()

	return newUploadAvatarRequestToPath(t, avatarRoutePath, userID, fileName, content)
}

func newUploadAvatarRequestToPath(
	t *testing.T,
	path string,
	userID string,
	fileName string,
	content []byte,
) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile(avatarFormFileField, fileName)
	require.NoError(t, err)
	_, err = file.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-User-ID", userID)

	return request
}

func newUploadAvatarRequestWithoutFile(t *testing.T, userID string) *http.Request {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("ignored", "value"))
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, avatarRoutePath, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-User-ID", userID)

	return request
}

func pngContent() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47,
		0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52,
	}
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, wantError string) {
	t.Helper()

	var body errorResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, wantError, body.Error)
}

func mustAvatarURL(t *testing.T, avatarID string) string {
	t.Helper()

	avatarURL, err := url.JoinPath(avatarRoutePath, avatarID)
	require.NoError(t, err)

	return avatarURL
}

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
