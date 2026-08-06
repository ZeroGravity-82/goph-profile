package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
	"github.com/ZeroGravity-82/goph-profile/web"
)

// TestAvatarHandler_getCurrentAvatarByEmail проверяет успешное получение текущей аватарки по email.
func TestAvatarHandler_getCurrentAvatarByEmail(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		currentByEmailOutput: usecase.GetCurrentAvatarByEmailOutput{
			Content:  pngContent(),
			MIMEType: model.MIMEPNG,
		},
	}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustCurrentAvatarByEmailURL(t, "  User@Example.COM  "), nil)
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByEmail(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, model.MIMEPNG, response.Header().Get("Content-Type"))
	assert.Equal(t, avatarCacheControl, response.Header().Get("Cache-Control"))
	assert.Equal(t, pngContent(), response.Body.Bytes())
	require.Len(t, avatarUseCase.currentByEmailInputs, 1)
	assert.Equal(t, model.Email("user@example.com"), avatarUseCase.currentByEmailInputs[0].Email)
}

// TestAvatarHandler_getCurrentAvatarByEmail_ReturnsDefaultAvatar проверяет выдачу PNG-заглушки по email.
func TestAvatarHandler_getCurrentAvatarByEmail_ReturnsDefaultAvatar(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		currentByEmailOutput: usecase.GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true},
	}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustCurrentAvatarByEmailURL(t, "user@example.com"), nil)
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByEmail(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, model.MIMEPNG, response.Header().Get("Content-Type"))
	assert.Equal(t, avatarCacheControl, response.Header().Get("Cache-Control"))
	assert.Equal(t, defaultAvatarPath, response.Header().Get(xAccelRedirectHeader))
	assert.NotEmpty(t, response.Body.Bytes())
	assert.Equal(t, web.DefaultAvatarPNG, response.Body.Bytes())
	require.Len(t, avatarUseCase.currentByEmailInputs, 1)
}

// TestAvatarHandler_getCurrentAvatarByEmail_RejectsInvalidEmail проверяет ошибку невалидного email.
func TestAvatarHandler_getCurrentAvatarByEmail_RejectsInvalidEmail(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/avatar", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByEmail(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.currentByEmailInputs)
	assertErrorResponse(t, response, "Invalid email")
}

// TestAvatarHandler_getCurrentAvatarByEmail_ReturnsInternalServerError проверяет внутреннюю ошибку получения аватарки.
func TestAvatarHandler_getCurrentAvatarByEmail_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{currentByEmailErr: errors.New("storage error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustCurrentAvatarByEmailURL(t, "user@example.com"), nil)
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByEmail(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.currentByEmailInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}

// TestAvatarHandler_getCurrentAvatarByUserID проверяет успешное получение текущей аватарки по ID пользователя.
func TestAvatarHandler_getCurrentAvatarByUserID(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		currentByUserOutput: usecase.GetCurrentAvatarByUserIDOutput{
			Content:  pngContent(),
			MIMEType: model.MIMEPNG,
		},
	}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newCurrentAvatarByUserIDRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByUserID(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, model.MIMEPNG, response.Header().Get("Content-Type"))
	assert.Equal(t, avatarCacheControl, response.Header().Get("Cache-Control"))
	assert.Equal(t, pngContent(), response.Body.Bytes())
	require.Len(t, avatarUseCase.currentByUserInputs, 1)
	assert.Equal(t, testUserID, avatarUseCase.currentByUserInputs[0].UserID)
}

// TestAvatarHandler_getCurrentAvatarByUserID_ReturnsDefaultAvatar проверяет выдачу PNG-заглушки по ID пользователя.
func TestAvatarHandler_getCurrentAvatarByUserID_ReturnsDefaultAvatar(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		currentByUserOutput: usecase.GetCurrentAvatarByUserIDOutput{UseDefaultAvatar: true},
	}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newCurrentAvatarByUserIDRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByUserID(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, model.MIMEPNG, response.Header().Get("Content-Type"))
	assert.Equal(t, avatarCacheControl, response.Header().Get("Cache-Control"))
	assert.Equal(t, defaultAvatarPath, response.Header().Get(xAccelRedirectHeader))
	assert.NotEmpty(t, response.Body.Bytes())
	assert.Equal(t, web.DefaultAvatarPNG, response.Body.Bytes())
	require.Len(t, avatarUseCase.currentByUserInputs, 1)
}

// TestAvatarHandler_getCurrentAvatarByUserID_RejectsInvalidUserID проверяет ошибку невалидного user_id в пути.
func TestAvatarHandler_getCurrentAvatarByUserID_RejectsInvalidUserID(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newCurrentAvatarByUserIDRequest(t, "not-a-uuid")
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByUserID(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.currentByUserInputs)
	assertErrorResponse(t, response, "Invalid user_id")
}

// TestAvatarHandler_getCurrentAvatarByUserID_ReturnsInternalServerError проверяет внутреннюю ошибку получения аватарки.
func TestAvatarHandler_getCurrentAvatarByUserID_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{currentByUserErr: errors.New("storage error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newCurrentAvatarByUserIDRequest(t, testUserID.String())
	response := httptest.NewRecorder()

	// Act
	handler.getCurrentAvatarByUserID(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.currentByUserInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
