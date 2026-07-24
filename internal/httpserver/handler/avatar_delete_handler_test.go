package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// TestAvatarHandler_deleteAvatar проверяет успешное удаление аватарки по ID.
func TestAvatarHandler_deleteAvatar(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteAvatar(response, request)

	// Assert
	require.Equal(t, http.StatusNoContent, response.Code)
	assert.Empty(t, response.Body.String())
	require.Len(t, avatarUseCase.deleteAvatarInputs, 1)
	assert.Equal(t, testUserID, avatarUseCase.deleteAvatarInputs[0].UserID)
	assert.Equal(t, testAvatarID, avatarUseCase.deleteAvatarInputs[0].AvatarID)
}

// TestAvatarHandler_deleteAvatar_RejectsInvalidUserIDHeader проверяет ошибку невалидного заголовка X-User-ID.
func TestAvatarHandler_deleteAvatar_RejectsInvalidUserIDHeader(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteAvatarRequest(t, "not-a-uuid", testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.deleteAvatarInputs)
	assertErrorResponse(t, response, "Invalid X-User-ID header")
}

// TestAvatarHandler_deleteAvatar_RejectsInvalidAvatarID проверяет ошибку невалидного avatar_id в пути.
func TestAvatarHandler_deleteAvatar_RejectsInvalidAvatarID(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteAvatarRequest(t, testUserID.String(), "not-a-uuid")
	response := httptest.NewRecorder()

	// Act
	handler.deleteAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.deleteAvatarInputs)
	assertErrorResponse(t, response, "Invalid avatar_id")
}

// TestAvatarHandler_deleteAvatar_ReturnsUserNotFound проверяет ошибку отсутствующего пользователя.
func TestAvatarHandler_deleteAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteAvatarErr: repository.ErrUserNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.deleteAvatarInputs, 1)
	assertErrorResponse(t, response, "User not found")
}

// TestAvatarHandler_deleteAvatar_ReturnsAvatarNotFound проверяет ошибку отсутствующей аватарки.
func TestAvatarHandler_deleteAvatar_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteAvatarErr: repository.ErrAvatarNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.deleteAvatarInputs, 1)
	assertErrorResponse(t, response, "Avatar not found")
}

// TestAvatarHandler_deleteAvatar_ReturnsAvatarForbidden проверяет ошибку удаления чужой аватарки.
func TestAvatarHandler_deleteAvatar_ReturnsAvatarForbidden(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteAvatarErr: model.ErrAvatarForbidden}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusForbidden, response.Code)
	require.Len(t, avatarUseCase.deleteAvatarInputs, 1)
	assertErrorResponse(t, response, "Forbidden")
}

// TestAvatarHandler_deleteAvatar_ReturnsInternalServerError проверяет внутреннюю ошибку удаления аватарки.
func TestAvatarHandler_deleteAvatar_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteAvatarErr: errors.New("database error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.deleteAvatarInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
