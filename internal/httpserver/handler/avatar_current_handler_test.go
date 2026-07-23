package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// TestAvatarHandler_selectCurrentAvatar проверяет успешный выбор текущей аватарки.
func TestAvatarHandler_selectCurrentAvatar(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	require.Equal(t, http.StatusNoContent, response.Code)
	assert.Empty(t, response.Body.String())
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assert.Equal(t, testUserID, avatarUseCase.selectCurrentInputs[0].UserID)
	assert.Equal(t, testAvatarID, avatarUseCase.selectCurrentInputs[0].AvatarID)
}

// TestAvatarHandler_selectCurrentAvatar_RejectsInvalidUserIDHeader проверяет ошибку невалидного заголовка X-User-ID.
func TestAvatarHandler_selectCurrentAvatar_RejectsInvalidUserIDHeader(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, "not-a-uuid", testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.selectCurrentInputs)
	assertErrorResponse(t, response, "Invalid X-User-ID header")
}

// TestAvatarHandler_selectCurrentAvatar_RejectsInvalidRequestBody проверяет ошибку невалидного JSON-тела.
func TestAvatarHandler_selectCurrentAvatar_RejectsInvalidRequestBody(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequestWithBody(
		t,
		testUserID.String(),
		bytes.NewBufferString("{"),
	)
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.selectCurrentInputs)
	assertErrorResponse(t, response, "Invalid request body")
}

// TestAvatarHandler_selectCurrentAvatar_RejectsInvalidAvatarID проверяет ошибку невалидного avatar_id в теле запроса.
func TestAvatarHandler_selectCurrentAvatar_RejectsInvalidAvatarID(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), "not-a-uuid")
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.selectCurrentInputs)
	assertErrorResponse(t, response, "Invalid avatar_id")
}

// TestAvatarHandler_selectCurrentAvatar_ReturnsUserNotFound проверяет ошибку отсутствующего пользователя.
func TestAvatarHandler_selectCurrentAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{selectCurrentErr: repository.ErrUserNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assertErrorResponse(t, response, "User not found")
}

// TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarNotFound проверяет ошибку отсутствующей аватарки.
func TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{selectCurrentErr: repository.ErrAvatarNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assertErrorResponse(t, response, "Avatar not found")
}

// TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarForbidden проверяет ошибку выбора чужой аватарки.
func TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarForbidden(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{selectCurrentErr: model.ErrAvatarForbidden}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusForbidden, response.Code)
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assertErrorResponse(t, response, "Forbidden")
}

// TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarNotReady проверяет ошибку выбора неготовой аватарки.
func TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarNotReady(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{selectCurrentErr: model.ErrAvatarNotReady}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusConflict, response.Code)
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assertErrorResponse(t, response, "Avatar is not ready")
}

// TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarDeleted проверяет ошибку выбора удаленной аватарки.
func TestAvatarHandler_selectCurrentAvatar_ReturnsAvatarDeleted(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{selectCurrentErr: model.ErrAvatarDeleted}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusConflict, response.Code)
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assertErrorResponse(t, response, "Avatar is deleted")
}

// TestAvatarHandler_selectCurrentAvatar_ReturnsInternalServerError проверяет внутреннюю ошибку выбора текущей аватарки.
func TestAvatarHandler_selectCurrentAvatar_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{selectCurrentErr: errors.New("database error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	handler.selectCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}

// TestAvatarHandler_deleteCurrentAvatar проверяет успешное удаление текущей аватарки.
func TestAvatarHandler_deleteCurrentAvatar(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteCurrentAvatarRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteCurrentAvatar(response, request)

	// Assert
	require.Equal(t, http.StatusNoContent, response.Code)
	assert.Empty(t, response.Body.String())
	require.Len(t, avatarUseCase.deleteCurrentInputs, 1)
	assert.Equal(t, testUserID, avatarUseCase.deleteCurrentInputs[0].UserID)
}

// TestAvatarHandler_deleteCurrentAvatar_RejectsInvalidUserIDHeader проверяет ошибку невалидного заголовка X-User-ID.
func TestAvatarHandler_deleteCurrentAvatar_RejectsInvalidUserIDHeader(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteCurrentAvatarRequest(t, "not-a-uuid")
	response := httptest.NewRecorder()

	// Act
	handler.deleteCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.deleteCurrentInputs)
	assertErrorResponse(t, response, "Invalid X-User-ID header")
}

// TestAvatarHandler_deleteCurrentAvatar_ReturnsUserNotFound проверяет ошибку отсутствующего пользователя.
func TestAvatarHandler_deleteCurrentAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteCurrentErr: repository.ErrUserNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteCurrentAvatarRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.deleteCurrentInputs, 1)
	assertErrorResponse(t, response, "User not found")
}

// TestAvatarHandler_deleteCurrentAvatar_ReturnsAvatarNotFound проверяет ошибку отсутствующей текущей аватарки.
func TestAvatarHandler_deleteCurrentAvatar_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteCurrentErr: repository.ErrAvatarNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteCurrentAvatarRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.deleteCurrentInputs, 1)
	assertErrorResponse(t, response, "Avatar not found")
}

// TestAvatarHandler_deleteCurrentAvatar_ReturnsAvatarForbidden проверяет ошибку удаления чужой аватарки.
func TestAvatarHandler_deleteCurrentAvatar_ReturnsAvatarForbidden(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteCurrentErr: model.ErrAvatarForbidden}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteCurrentAvatarRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusForbidden, response.Code)
	require.Len(t, avatarUseCase.deleteCurrentInputs, 1)
	assertErrorResponse(t, response, "Forbidden")
}

// TestAvatarHandler_deleteCurrentAvatar_ReturnsInternalServerError проверяет внутреннюю ошибку удаления текущей
// аватарки.
func TestAvatarHandler_deleteCurrentAvatar_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{deleteCurrentErr: errors.New("database error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newDeleteCurrentAvatarRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.deleteCurrentAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.deleteCurrentInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
