package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestNewRouter_UploadAvatarRoute проверяет регистрацию ручки загрузки аватарки.
func TestNewRouter_UploadAvatarRoute(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, avatarUseCase.uploadInputs, 1)
}

// TestNewRouter_UploadAvatarRoute_WithNilLogger проверяет работу роутера без переданного логгера.
func TestNewRouter_UploadAvatarRoute_WithNilLogger(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, nil)
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, avatarUseCase.uploadInputs, 1)
}

// TestNewRouter_UploadAvatarRoute_StripsTrailingSlash проверяет нормализацию завершающего слеша в ручке загрузки
// аватарки.
func TestNewRouter_UploadAvatarRoute_StripsTrailingSlash(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, discardLogger())
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
	require.Len(t, avatarUseCase.uploadInputs, 1)
}

// TestNewRouter_ReturnsMethodNotAllowed проверяет ошибку неподдержанного HTTP-метода.
func TestNewRouter_ReturnsMethodNotAllowed(t *testing.T) {
	// Arrange
	router := NewRouter(&avatarUseCaseFake{}, &userUseCaseFake{}, discardLogger())
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
	router := NewRouter(&avatarUseCaseFake{}, &userUseCaseFake{}, discardLogger())
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
	avatarUseCase := &avatarUseCaseFake{}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, discardLogger())
	request := newUploadAvatarRequest(t, testUserID.String(), "avatar.png", pngContent())
	gzipRequestBody(t, request)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, avatarUseCase.uploadInputs, 1)
}

// TestNewRouter_SelectCurrentAvatarRoute проверяет регистрацию ручки выбора текущей аватарки.
func TestNewRouter_SelectCurrentAvatarRoute(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, discardLogger())
	request := newSelectCurrentAvatarRequest(t, testUserID.String(), testAvatarID.String())
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Len(t, avatarUseCase.selectCurrentInputs, 1)
	assert.Equal(t, testUserID, avatarUseCase.selectCurrentInputs[0].UserID)
	assert.Equal(t, testAvatarID, avatarUseCase.selectCurrentInputs[0].AvatarID)
}

// TestNewRouter_AvatarMetadataRoute проверяет регистрацию ручки получения метаданных аватарки.
func TestNewRouter_AvatarMetadataRoute(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustAvatarMetadataURL(t, testAvatarID.String()), nil)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, avatarUseCase.metadataInputs, 1)
	assert.Equal(t, testAvatarID, avatarUseCase.metadataInputs[0].AvatarID)
}

// TestNewRouter_GetAvatarRoute проверяет регистрацию ручки получения файла аватарки.
func TestNewRouter_GetAvatarRoute(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		getAvatarOutput: usecase.GetAvatarOutput{
			Content:  pngContent(),
			MIMEType: model.MIMEPNG,
		},
	}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustAvatarURL(t, testAvatarID.String()), nil)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, avatarUseCase.getAvatarInputs, 1)
	assert.Equal(t, testAvatarID, avatarUseCase.getAvatarInputs[0].AvatarID)
}

// TestNewRouter_PublicAvatarRoute проверяет регистрацию ручки публичного получения аватарки по email.
func TestNewRouter_PublicAvatarRoute(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		currentByEmailOutput: usecase.GetCurrentAvatarByEmailOutput{
			Content:  pngContent(),
			MIMEType: model.MIMEPNG,
		},
	}
	router := NewRouter(avatarUseCase, &userUseCaseFake{}, discardLogger())
	request := httptest.NewRequest(http.MethodGet, mustPublicAvatarURL(t, "user@example.com"), nil)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, avatarUseCase.currentByEmailInputs, 1)
	assert.Equal(t, model.Email("user@example.com"), avatarUseCase.currentByEmailInputs[0].Email)
}

// TestNewRouter_UserResolveRoute проверяет регистрацию ручки определения пользователя по email.
func TestNewRouter_UserResolveRoute(t *testing.T) {
	// Arrange
	userUseCase := &userUseCaseFake{
		resolveOutput: usecase.ResolveUserByEmailOutput{
			ID:    testUserID,
			Email: model.Email("user@example.com"),
		},
	}
	router := NewRouter(&avatarUseCaseFake{}, userUseCase, discardLogger())
	request := newResolveUserRequest(t, "user@example.com")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, []model.Email{"user@example.com"}, userUseCase.resolveInputs)
}

// TestNewRouter_UserResolveRoute_ReturnsUnsupportedMediaType проверяет ошибку неподдерживаемого content-type.
func TestNewRouter_UserResolveRoute_ReturnsUnsupportedMediaType(t *testing.T) {
	// Arrange
	userUseCase := &userUseCaseFake{}
	router := NewRouter(&avatarUseCaseFake{}, userUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodPost, userResolveRoutePath, bytes.NewBufferString("email=user@example.com"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
	assert.Empty(t, userUseCase.resolveInputs)
}
