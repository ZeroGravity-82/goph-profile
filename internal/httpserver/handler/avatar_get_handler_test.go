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
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestAvatarHandler_getAvatar проверяет успешное получение файла аватарки по ID.
func TestAvatarHandler_getAvatar(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		getAvatarOutput: usecase.GetAvatarOutput{
			Content:  pngContent(),
			MIMEType: model.MIMEPNG,
		},
	}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, testAvatarID.String(), "100x100", "png")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, model.MIMEPNG, response.Header().Get("Content-Type"))
	assert.Equal(t, avatarCacheControl, response.Header().Get("Cache-Control"))
	assert.Equal(t, pngContent(), response.Body.Bytes())
	require.Len(t, avatarUseCase.getAvatarInputs, 1)
	assert.Equal(t, testAvatarID, avatarUseCase.getAvatarInputs[0].AvatarID)
	assert.Equal(t, usecase.AvatarSize100, avatarUseCase.getAvatarInputs[0].Size)
	assert.Equal(t, model.MIMEPNG, avatarUseCase.getAvatarInputs[0].MIMEType)
}

// TestAvatarHandler_getAvatar_UsesOriginalSizeByDefault проверяет получение оригинала без query-параметра size.
func TestAvatarHandler_getAvatar_UsesOriginalSizeByDefault(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{
		getAvatarOutput: usecase.GetAvatarOutput{
			Content:  pngContent(),
			MIMEType: model.MIMEPNG,
		},
	}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, testAvatarID.String(), "", "")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, avatarUseCase.getAvatarInputs, 1)
	assert.Equal(t, usecase.AvatarSizeOriginal, avatarUseCase.getAvatarInputs[0].Size)
	assert.Empty(t, avatarUseCase.getAvatarInputs[0].MIMEType)
}

// TestAvatarHandler_getAvatar_RejectsInvalidAvatarID проверяет ошибку невалидного avatar_id в пути.
func TestAvatarHandler_getAvatar_RejectsInvalidAvatarID(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, "not-a-uuid", "", "")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.getAvatarInputs)
	assertErrorResponse(t, response, "Invalid avatar_id")
}

// TestAvatarHandler_getAvatar_RejectsInvalidSize проверяет ошибку неподдерживаемого размера аватарки.
func TestAvatarHandler_getAvatar_RejectsInvalidSize(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, testAvatarID.String(), "200x200", "")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.getAvatarInputs)
	assertErrorResponse(t, response, "Invalid size")
}

// TestAvatarHandler_getAvatar_RejectsInvalidFormat проверяет ошибку неподдерживаемого формата аватарки.
func TestAvatarHandler_getAvatar_RejectsInvalidFormat(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, testAvatarID.String(), "", "gif")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, avatarUseCase.getAvatarInputs)
	assertErrorResponse(t, response, "Invalid format")
}

// TestAvatarHandler_getAvatar_ReturnsAvatarNotFound проверяет ошибку отсутствия аватарки.
func TestAvatarHandler_getAvatar_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{getAvatarErr: repository.ErrAvatarNotFound}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, testAvatarID.String(), "", "")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	require.Len(t, avatarUseCase.getAvatarInputs, 1)
	assertErrorResponse(t, response, "Avatar not found")
}

// TestAvatarHandler_getAvatar_ReturnsFormatMismatch проверяет ошибку несовпадения запрошенного формата.
func TestAvatarHandler_getAvatar_ReturnsFormatMismatch(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{getAvatarErr: model.ErrInvalidAvatarMetadata}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, testAvatarID.String(), "", "jpeg")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	require.Len(t, avatarUseCase.getAvatarInputs, 1)
	assertErrorResponse(t, response, "Invalid format")
}

// TestAvatarHandler_getAvatar_ReturnsInternalServerError проверяет внутреннюю ошибку получения аватарки.
func TestAvatarHandler_getAvatar_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	avatarUseCase := &avatarUseCaseFake{getAvatarErr: errors.New("storage error")}
	handler := mustAvatarHandler(t, avatarUseCase, discardLogger())
	request := newGetAvatarRequest(t, testAvatarID.String(), "", "")
	response := httptest.NewRecorder()

	// Act
	handler.getAvatar(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, avatarUseCase.getAvatarInputs, 1)
	assertErrorResponse(t, response, "Internal server error")
}
