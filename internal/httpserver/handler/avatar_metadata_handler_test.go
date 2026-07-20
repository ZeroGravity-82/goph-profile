package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

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
	avatarUseCase := &avatarUseCaseFake{metadataErr: errors.New("database error")}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := newGetAvatarMetadataRequest(t, testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.getAvatarMetadata(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.metadataInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
