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
)

// TestAvatarHandler_getPublicAvatarByEmail проверяет успешное получение текущей аватарки по email.
func TestAvatarHandler_getPublicAvatarByEmail(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		currentByEmailOutput: usecase.GetCurrentAvatarByEmailOutput{
			Content:  pngContent(),
			MIMEType: model.MIMEPNG,
		},
	}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustPublicAvatarURL(t, "  User@Example.COM  "), nil)
	response := httptest.NewRecorder()

	// Act
	handler.getPublicAvatarByEmail(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, model.MIMEPNG, response.Header().Get("Content-Type"))
	assert.Equal(t, publicAvatarCacheControl, response.Header().Get("Cache-Control"))
	assert.Equal(t, pngContent(), response.Body.Bytes())
	require.Len(t, avatarUseCase.currentByEmailInputs, 1)
	assert.Equal(t, model.Email("user@example.com"), avatarUseCase.currentByEmailInputs[0].Email)
}

// TestAvatarHandler_getPublicAvatarByEmail_ReturnsDefaultAvatar проверяет выдачу PNG-заглушки.
func TestAvatarHandler_getPublicAvatarByEmail_ReturnsDefaultAvatar(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		currentByEmailOutput: usecase.GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true},
	}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustPublicAvatarURL(t, "user@example.com"), nil)
	response := httptest.NewRecorder()

	// Act
	handler.getPublicAvatarByEmail(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, model.MIMEPNG, response.Header().Get("Content-Type"))
	assert.Equal(t, publicAvatarCacheControl, response.Header().Get("Cache-Control"))
	assert.Equal(t, defaultAvatarPNG, response.Body.Bytes())
	require.Len(t, avatarUseCase.currentByEmailInputs, 1)
}

// TestAvatarHandler_getPublicAvatarByEmail_RejectsInvalidEmail проверяет ошибку невалидного email.
func TestAvatarHandler_getPublicAvatarByEmail_RejectsInvalidEmail(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/avatar", nil)
	response := httptest.NewRecorder()

	// Act
	handler.getPublicAvatarByEmail(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.currentByEmailInputs)
	assertErrorResponse(t, response, "Invalid email")
}

// TestAvatarHandler_getPublicAvatarByEmail_ReturnsInternalServerError проверяет внутреннюю ошибку получения аватарки.
func TestAvatarHandler_getPublicAvatarByEmail_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{currentByEmailErr: errors.New("storage error")}
	handler := NewAvatarHandler(avatarUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustPublicAvatarURL(t, "user@example.com"), nil)
	response := httptest.NewRecorder()

	// Act
	handler.getPublicAvatarByEmail(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.currentByEmailInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
