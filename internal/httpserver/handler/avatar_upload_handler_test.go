package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

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
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
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
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
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
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
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
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.txt", []byte("not an image"))
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.uploadInputs)
	assertErrorResponse(t, response, "Invalid file format")
}

// TestAvatarHandler_uploadAvatar_RejectsTooLongFileName проверяет ошибку слишком длинного имени файла.
func TestAvatarHandler_uploadAvatar_RejectsTooLongFileName(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	fileName := strings.Repeat("a", maxAvatarFileNameLengthBytes+1) + ".png"
	request := newUploadAvatarRequest(t, testUserID.String(), fileName, pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.uploadInputs)
	assertErrorResponse(t, response, "Invalid file name")
}

// TestAvatarHandler_uploadAvatar_RejectsTooLargeFile проверяет ошибку слишком большого файла.
func TestAvatarHandler_uploadAvatar_RejectsTooLargeFile(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
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

// TestAvatarHandler_uploadAvatar_RejectsTooLargeImage проверяет ошибку слишком большого изображения.
func TestAvatarHandler_uploadAvatar_RejectsTooLargeImage(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	content := pngContentWithSize(maxAvatarImageWidthPixels+1, 100)
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", content)
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.uploadInputs)
	assertErrorResponse(t, response, "Image dimensions are too large")
}

// TestAvatarHandler_uploadAvatar_ReturnsUserNotFound проверяет ошибку отсутствующего пользователя.
func TestAvatarHandler_uploadAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{uploadErr: usecase.ErrUserNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
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
	avatarUseCase := &avatarUseCaseFake{uploadErr: errors.New("database error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	handler.uploadAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.uploadInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
