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
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestAvatarHandler_listUserAvatars проверяет успешное получение списка аватарок пользователя.
func TestAvatarHandler_listUserAvatars(t *testing.T) {
	// Arrange
	createdAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 7, 18, 12, 0, 5, 0, time.UTC)
	width := 1920
	height := 1080
	avatarUseCase := &avatarUseCaseFake{
		listAvatarsOutput: usecase.ListUserAvatarsOutput{
			Avatars: []usecase.ListUserAvatarsItemOutput{
				{
					ID:        testAvatarID,
					UserID:    testUserID,
					FileName:  "avatar.jpg",
					MIMEType:  model.MIMEJPEG,
					SizeBytes: 1024000,
					Width:     &width,
					Height:    &height,
					Status:    model.AvatarStatusReady,
					IsCurrent: true,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			},
		},
	}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newListUserAvatarsRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.listUserAvatars(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	require.Len(t, avatarUseCase.listAvatarsInputs, 1)
	assert.Equal(t, testUserID, avatarUseCase.listAvatarsInputs[0].UserID)

	var body dto.ListUserAvatarsResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Len(t, body.Avatars, 1)
	assert.Equal(t, testAvatarID.String(), body.Avatars[0].ID)
	assert.Equal(t, testUserID.String(), body.Avatars[0].UserID)
	assert.Equal(t, mustAvatarURL(t, testAvatarID.String()), body.Avatars[0].URL)
	assert.Equal(t, "avatar.jpg", body.Avatars[0].FileName)
	assert.Equal(t, model.MIMEJPEG, body.Avatars[0].MIMEType)
	assert.Equal(t, int64(1024000), body.Avatars[0].SizeBytes)
	require.NotNil(t, body.Avatars[0].Width)
	assert.Equal(t, width, *body.Avatars[0].Width)
	require.NotNil(t, body.Avatars[0].Height)
	assert.Equal(t, height, *body.Avatars[0].Height)
	assert.Equal(t, string(model.AvatarStatusReady), body.Avatars[0].Status)
	assert.True(t, body.Avatars[0].IsCurrent)
	assert.Equal(t, createdAt, body.Avatars[0].CreatedAt)
	assert.Equal(t, updatedAt, body.Avatars[0].UpdatedAt)
}

// TestAvatarHandler_listUserAvatars_RejectsInvalidUserID проверяет ошибку невалидного user_id в пути.
func TestAvatarHandler_listUserAvatars_RejectsInvalidUserID(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newListUserAvatarsRequest(t, "not-a-uuid")
	response := httptest.NewRecorder()

	// Act
	handler.listUserAvatars(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.listAvatarsInputs)
	assertErrorResponse(t, response, "Invalid user_id")
}

// TestAvatarHandler_listUserAvatars_ReturnsUserNotFound проверяет ошибку отсутствия пользователя.
func TestAvatarHandler_listUserAvatars_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{listAvatarsErr: repository.ErrUserNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newListUserAvatarsRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.listUserAvatars(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.listAvatarsInputs, 1)
	assertErrorResponse(t, response, "User not found")
}

// TestAvatarHandler_listUserAvatars_ReturnsInternalServerError проверяет внутреннюю ошибку получения списка.
func TestAvatarHandler_listUserAvatars_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{listAvatarsErr: errors.New("database error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newListUserAvatarsRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.listUserAvatars(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.listAvatarsInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
