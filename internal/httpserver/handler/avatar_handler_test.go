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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

var (
	testUserID   = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001")
	testAvatarID = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")
)

type avatarUseCaseFake struct {
	uploadOutput   usecase.UploadAvatarOutput
	uploadErr      error
	uploadInputs   []usecase.UploadAvatarInput
	metadataOutput usecase.GetAvatarMetadataOutput
	metadataErr    error
	metadataInputs []usecase.GetAvatarMetadataInput
}

func (uc *avatarUseCaseFake) UploadAvatar(
	_ context.Context,
	input usecase.UploadAvatarInput,
) (usecase.UploadAvatarOutput, error) {
	uc.uploadInputs = append(uc.uploadInputs, input)
	return uc.uploadOutput, uc.uploadErr
}

func (uc *avatarUseCaseFake) GetAvatarMetadata(
	_ context.Context,
	input usecase.GetAvatarMetadataInput,
) (usecase.GetAvatarMetadataOutput, error) {
	uc.metadataInputs = append(uc.metadataInputs, input)
	return uc.metadataOutput, uc.metadataErr
}

// TestAvatarHandler_uploadAvatar проверяет успешную загрузку аватарки.
func TestAvatarHandler_uploadAvatar(t *testing.T) {
	// Arrange
	createdAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatarUseCase := &avatarUseCaseFake{
		uploadOutput: usecase.UploadAvatarOutput{
			ID:        testAvatarID,
			UserID:    testUserID,
			Status:    model.AvatarStatusProcessing,
			CreatedAt: createdAt,
		},
	}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	require.Len(t, avatarUseCase.uploadInputs, 1)
	assert.Equal(t, testUserID, avatarUseCase.uploadInputs[0].UserID)
	assert.Equal(t, "avatar.png", avatarUseCase.uploadInputs[0].FileName)
	assert.Equal(t, model.MIMEPNG, avatarUseCase.uploadInputs[0].MIMEType)
	assert.Equal(t, pngContent(), avatarUseCase.uploadInputs[0].Content)

	var body dto.UploadAvatarResponse
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
	avatarUseCase := &avatarUseCaseFake{}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newUploadAvatarRequest(t, "not-a-uuid", "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.uploadInputs)
	assertErrorResponse(t, response, "Invalid X-User-ID header")
}

// TestAvatarHandler_uploadAvatar_RejectsMissingFile проверяет ошибку отсутствующего файла.
func TestAvatarHandler_uploadAvatar_RejectsMissingFile(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newUploadAvatarRequestWithoutFile(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.uploadInputs)
	assertErrorResponse(t, response, "Invalid file format")
}

// TestAvatarHandler_uploadAvatar_RejectsUnsupportedFormat проверяет ошибку неподдерживаемого формата файла.
func TestAvatarHandler_uploadAvatar_RejectsUnsupportedFormat(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.txt", []byte("not an image"))
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.uploadInputs)
	assertErrorResponse(t, response, "Invalid file format")
}

// TestAvatarHandler_uploadAvatar_RejectsTooLargeFile проверяет ошибку слишком большого файла.
func TestAvatarHandler_uploadAvatar_RejectsTooLargeFile(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	content := bytes.Repeat([]byte{0x89}, maxAvatarFileSizeBytes+1)
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", content)
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	assert.Empty(t, avatarUseCase.uploadInputs)

	var body dto.ErrorResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "File too large", body.Error)
	assert.Equal(t, int64(maxAvatarFileSizeBytes), body.MaxSize)
}

// TestAvatarHandler_uploadAvatar_ReturnsUserNotFound проверяет ошибку отсутствующего пользователя.
func TestAvatarHandler_uploadAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{uploadErr: usecase.ErrUserNotFound}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.uploadInputs, 1)
	assertErrorResponse(t, response, "User not found")
}

// TestAvatarHandler_uploadAvatar_ReturnsInternalServerError проверяет внутреннюю ошибку загрузки аватарки.
func TestAvatarHandler_uploadAvatar_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	avatarUseCase := &avatarUseCaseFake{uploadErr: errors.New("database error")}
	handler := NewAvatarHandler(avatarUseCase, newTextLogger(logBuffer))
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.uploadInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
	assert.Contains(t, logBuffer.String(), "database error")
}

// TestAvatarHandler_getAvatarMetadata проверяет успешное получение метаданных аватарки.
func TestAvatarHandler_getAvatarMetadata(t *testing.T) {
	// Arrange
	createdAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 7, 18, 12, 0, 5, 0, time.UTC)
	width := 1920
	height := 1080
	thumb100 := "thumbs/avatar-id/100.png"
	thumb300 := "thumbs/avatar-id/300.png"
	avatarUseCase := &avatarUseCaseFake{
		metadataOutput: usecase.GetAvatarMetadataOutput{
			ID:                testAvatarID,
			UserID:            testUserID,
			FileName:          "avatar.jpg",
			MIMEType:          model.MIMEJPEG,
			SizeBytes:         1024000,
			Width:             &width,
			Height:            &height,
			ObjectKeyThumb100: &thumb100,
			ObjectKeyThumb300: &thumb300,
			Status:            model.AvatarStatusReady,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		},
	}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newGetAvatarMetadataRequest(t, testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.getAvatarMetadata(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	require.Len(t, avatarUseCase.metadataInputs, 1)
	assert.Equal(t, testAvatarID, avatarUseCase.metadataInputs[0].AvatarID)

	var body dto.AvatarMetadataResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, testAvatarID.String(), body.ID)
	assert.Equal(t, testUserID.String(), body.UserID)
	assert.Equal(t, "avatar.jpg", body.FileName)
	assert.Equal(t, model.MIMEJPEG, body.MIMEType)
	assert.Equal(t, int64(1024000), body.SizeBytes)
	require.NotNil(t, body.Width)
	assert.Equal(t, width, *body.Width)
	require.NotNil(t, body.Height)
	assert.Equal(t, height, *body.Height)
	assert.Equal(t, string(model.AvatarStatusReady), body.Status)
	assert.Equal(t, createdAt, body.CreatedAt)
	assert.Equal(t, updatedAt, body.UpdatedAt)
	assert.Equal(t, []dto.AvatarThumbnailResponse{
		{Size: "100x100", URL: mustAvatarThumbnailURL(t, testAvatarID, "100x100")},
		{Size: "300x300", URL: mustAvatarThumbnailURL(t, testAvatarID, "300x300")},
	}, body.Thumbnails)
}

// TestAvatarHandler_getAvatarMetadata_RejectsInvalidAvatarID проверяет ошибку невалидного avatar_id в пути.
func TestAvatarHandler_getAvatarMetadata_RejectsInvalidAvatarID(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newGetAvatarMetadataRequest(t, "not-a-uuid")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatarMetadata(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.metadataInputs)
	assertErrorResponse(t, response, "Invalid avatar_id")
}

// TestAvatarHandler_getAvatarMetadata_ReturnsAvatarNotFound проверяет ошибку отсутствия аватарки.
func TestAvatarHandler_getAvatarMetadata_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{metadataErr: usecase.ErrAvatarNotFound}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newGetAvatarMetadataRequest(t, testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.getAvatarMetadata(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.metadataInputs, 1)
	assertErrorResponse(t, response, "Avatar not found")
}

// TestAvatarHandler_getAvatarMetadata_ReturnsInternalServerError проверяет внутреннюю ошибку получения метаданных.
func TestAvatarHandler_getAvatarMetadata_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	avatarUseCase := &avatarUseCaseFake{metadataErr: errors.New("database error")}
	handler := NewAvatarHandler(avatarUseCase, newTextLogger(logBuffer))
	request := newGetAvatarMetadataRequest(t, testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.getAvatarMetadata(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.metadataInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
	assert.Contains(t, logBuffer.String(), "database error")
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
	file, err := writer.CreateFormFile(formFileField, fileName)
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

func newGetAvatarMetadataRequest(t *testing.T, avatarID string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, mustAvatarMetadataURL(t, avatarID), nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(avatarIDRouteParam, avatarID)

	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func pngContent() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47,
		0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52,
	}
}

func mustAvatarURL(t *testing.T, avatarID string) string {
	t.Helper()

	avatarURL, err := url.JoinPath(avatarRoutePath, avatarID)
	require.NoError(t, err)

	return avatarURL
}

func mustAvatarThumbnailURL(t *testing.T, avatarID uuid.UUID, size string) string {
	t.Helper()

	thumbnailURL, err := avatarURLForID(avatarID)
	require.NoError(t, err)
	values := url.Values{}
	values.Set("size", size)

	return thumbnailURL + "?" + values.Encode()
}
