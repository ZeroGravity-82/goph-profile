package worker

import (
	"bytes"
	"context"
	stdimage "image"
	"image/color"
	"image/png"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

var (
	testUserID            = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001")
	testAvatarID          = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")
	testObjectKeyOriginal = "users/user-id/avatars/avatar-id/original"
	testObjectKeyThumb100 = "users/user-id/avatars/avatar-id/thumb-100.png"
	testObjectKeyThumb300 = "users/user-id/avatars/avatar-id/thumb-300.png"
)

// TestAvatarHandler_HandleAvatarProcessing проверяет создание миниатюр и завершение обработки аватарки.
func TestAvatarHandler_HandleAvatarProcessing(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileStorage := &fileStorageFake{content: testPNG(t, 320, 240)}
	useCase := &avatarWorkerUseCaseFake{}
	handler := mustAvatarHandler(t, useCase, fileStorage)
	message := usecase.AvatarProcessingMessage{
		AvatarID:          testAvatarID,
		UserID:            testUserID,
		ObjectKeyOriginal: testObjectKeyOriginal,
	}

	// Act
	err := handler.HandleAvatarProcessing(ctx, message)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []string{testObjectKeyOriginal}, fileStorage.gets)
	assert.Equal(t, []string{testObjectKeyThumb100, testObjectKeyThumb300}, fileStorage.putKeys())
	assertPNGSize(t, fileStorage.puts[0].content, 100, 100)
	assertPNGSize(t, fileStorage.puts[1].content, 300, 300)
	require.Len(t, useCase.readyInputs, 1)
	assert.Equal(t, testAvatarID, useCase.readyInputs[0].AvatarID)
	assert.Equal(t, 320, useCase.readyInputs[0].Width)
	assert.Equal(t, 240, useCase.readyInputs[0].Height)
	assert.Equal(t, testObjectKeyThumb100, useCase.readyInputs[0].ObjectKeyThumb100)
	assert.Equal(t, testObjectKeyThumb300, useCase.readyInputs[0].ObjectKeyThumb300)
	assert.Empty(t, useCase.failedInputs)
}

// testPNG создает PNG-изображение заданного размера для проверки обработки аватарок воркером.
func testPNG(t *testing.T, width int, height int) []byte {
	t.Helper()

	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 180, A: 255})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func mustAvatarHandler(t *testing.T, useCase avatarWorkerUseCase, fileStorage fileStorage) *AvatarHandler {
	t.Helper()

	handler, err := NewAvatarHandler(useCase, fileStorage, nil)
	require.NoError(t, err)
	return handler
}

// assertPNGSize проверяет размеры PNG-изображения без декодирования всех пикселей.
func assertPNGSize(t *testing.T, content []byte, width int, height int) {
	t.Helper()

	cfg, err := png.DecodeConfig(bytes.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, width, cfg.Width)
	assert.Equal(t, height, cfg.Height)
}

// TestAvatarHandler_HandleAvatarProcessing_MarksAvatarFailed проверяет завершение обработки ошибкой после битого файла.
func TestAvatarHandler_HandleAvatarProcessing_MarksAvatarFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileStorage := &fileStorageFake{content: []byte("not an image")}
	useCase := &avatarWorkerUseCaseFake{}
	handler := mustAvatarHandler(t, useCase, fileStorage)
	message := usecase.AvatarProcessingMessage{
		AvatarID:          testAvatarID,
		UserID:            testUserID,
		ObjectKeyOriginal: testObjectKeyOriginal,
	}

	// Act
	err := handler.HandleAvatarProcessing(ctx, message)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, fileStorage.puts)
	assert.Empty(t, useCase.readyInputs)
	assert.Equal(t, []usecase.MarkAvatarFailedInput{{AvatarID: testAvatarID}}, useCase.failedInputs)
}

// TestAvatarHandler_HandleAvatarDeletion проверяет удаление файлов аватарки и завершение удаления.
func TestAvatarHandler_HandleAvatarDeletion(t *testing.T) {
	// Arrange
	ctx := context.Background()
	fileStorage := &fileStorageFake{}
	useCase := &avatarWorkerUseCaseFake{}
	handler := mustAvatarHandler(t, useCase, fileStorage)
	message := usecase.AvatarDeletionMessage{
		AvatarID:   testAvatarID,
		ObjectKeys: []string{testObjectKeyOriginal, testObjectKeyThumb100, testObjectKeyThumb300},
	}

	// Act
	err := handler.HandleAvatarDeletion(ctx, message)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, message.ObjectKeys, fileStorage.deletes)
	assert.Equal(t, []usecase.MarkAvatarDeletedInput{{AvatarID: testAvatarID}}, useCase.deletedInputs)
}
