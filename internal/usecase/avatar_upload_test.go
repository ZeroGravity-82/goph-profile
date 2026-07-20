package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// TestAvatarUseCase_UploadAvatar проверяет успешную загрузку аватарки.
func TestAvatarUseCase_UploadAvatar(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	user := mustUseCaseUser(t, now)
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{}
	fileStore := &fileStoreFake{objectKey: testObjectKeyOriginal}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, messagePublisher)
	input := validUploadAvatarInput()

	// Act
	result, err := useCase.UploadAvatar(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), result.ID.Version())
	assert.Equal(t, testUserID, result.UserID)
	assert.Equal(t, model.AvatarStatusProcessing, result.Status)
	assert.Equal(t, int64(len(input.Content)), result.SizeBytes)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	require.Len(t, fileStore.objectKeyCalls, 1)
	assert.Equal(t, testUserID, fileStore.objectKeyCalls[0].userID)
	assert.Equal(t, result.ID, fileStore.objectKeyCalls[0].avatarID)
	require.Len(t, fileStore.puts, 1)
	assert.Equal(t, testObjectKeyOriginal, fileStore.puts[0].objectKey)
	assert.Equal(t, input.Content, fileStore.puts[0].content)
	require.Len(t, avatarRepo.created, 1)
	assert.Equal(t, result.ID, avatarRepo.created[0].ID)
	assert.Equal(t, testObjectKeyOriginal, avatarRepo.created[0].ObjectKeyOriginal)
	assert.Equal(t, []AvatarProcessingMessage{
		{AvatarID: result.ID, UserID: testUserID, ObjectKeyOriginal: testObjectKeyOriginal},
	}, messagePublisher.messages)
	assert.Empty(t, avatarRepo.deletedIDs)
	assert.Empty(t, fileStore.deletes)
}

func validUploadAvatarInput() UploadAvatarInput {
	return UploadAvatarInput{
		UserID:   testUserID,
		FileName: "avatar.png",
		MIMEType: model.MIMEPNG,
		Content:  []byte("image content"),
	}
}

// TestAvatarUseCase_UploadAvatar_RejectsInvalidMetadata проверяет ошибку невалидных метаданных при загрузке аватарки.
func TestAvatarUseCase_UploadAvatar_RejectsInvalidMetadata(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo := &avatarUserRepositoryFake{}
	avatarRepo := &avatarRepositoryFake{}
	fileStore := &fileStoreFake{objectKey: testObjectKeyOriginal}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, messagePublisher)
	input := validUploadAvatarInput()
	input.Content = nil

	// Act
	result, err := useCase.UploadAvatar(ctx, input)

	// Assert
	require.ErrorIs(t, err, model.ErrInvalidAvatarMetadata)
	assert.Zero(t, result)
	assert.Empty(t, userRepo.ids)
	assert.Empty(t, fileStore.puts)
	assert.Empty(t, avatarRepo.created)
	assert.Empty(t, messagePublisher.messages)
}

// TestAvatarUseCase_UploadAvatar_ReturnsUserNotFound проверяет ошибку отсутствия пользователя при загрузке аватарки.
func TestAvatarUseCase_UploadAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo := &avatarUserRepositoryFake{err: ErrUserNotFound}
	avatarRepo := &avatarRepositoryFake{}
	fileStore := &fileStoreFake{objectKey: testObjectKeyOriginal}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, messagePublisher)

	// Act
	result, err := useCase.UploadAvatar(ctx, validUploadAvatarInput())

	// Assert
	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Zero(t, result)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Empty(t, fileStore.puts)
	assert.Empty(t, avatarRepo.created)
	assert.Empty(t, messagePublisher.messages)
}

// TestAvatarUseCase_UploadAvatar_ReturnsPutOriginalObjectError проверяет ошибку сохранения исходного файла при загрузке
// аватарки.
func TestAvatarUseCase_UploadAvatar_ReturnsPutOriginalObjectError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	putErr := errors.New("put object error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{}
	fileStore := &fileStoreFake{objectKey: testObjectKeyOriginal, putErr: putErr}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, messagePublisher)

	// Act
	result, err := useCase.UploadAvatar(ctx, validUploadAvatarInput())

	// Assert
	require.ErrorIs(t, err, putErr)
	assert.Zero(t, result)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	require.Len(t, fileStore.puts, 1)
	assert.Equal(t, testObjectKeyOriginal, fileStore.puts[0].objectKey)
	assert.Empty(t, avatarRepo.created)
	assert.Empty(t, messagePublisher.messages)
	assert.Empty(t, fileStore.deletes)
}

// TestAvatarUseCase_UploadAvatar_DeletesOriginalObjectAfterCreateError проверяет удаление исходного файла из
// хранилища после ошибки создания записи в БД при загрузке аватарки.
func TestAvatarUseCase_UploadAvatar_DeletesOriginalObjectAfterCreateError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	createErr := errors.New("create error")
	deleteErr := errors.New("delete object error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{createErr: createErr}
	fileStore := &fileStoreFake{objectKey: testObjectKeyOriginal, deleteErr: deleteErr}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, messagePublisher)

	// Act
	result, err := useCase.UploadAvatar(ctx, validUploadAvatarInput())

	// Assert
	require.ErrorIs(t, err, createErr)
	require.ErrorIs(t, err, deleteErr)
	assert.Zero(t, result)
	assert.Equal(t, []string{testObjectKeyOriginal}, fileStore.deletes)
	assert.Empty(t, messagePublisher.messages)
}

// TestAvatarUseCase_UploadAvatar_DeletesAvatarAndOriginalObjectAfterPublishError проверяет удаление записи в БД и
// исходного файла после ошибки публикации сообщения о загрузке аватарки.
func TestAvatarUseCase_UploadAvatar_DeletesAvatarAndOriginalObjectAfterPublishError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	publishErr := errors.New("publish error")
	deleteAvatarErr := errors.New("delete avatar error")
	deleteObjectErr := errors.New("delete object error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{deleteErr: deleteAvatarErr}
	fileStore := &fileStoreFake{objectKey: testObjectKeyOriginal, deleteErr: deleteObjectErr}
	messagePublisher := &avatarMessagePublisherFake{err: publishErr}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, messagePublisher)

	// Act
	result, err := useCase.UploadAvatar(ctx, validUploadAvatarInput())

	// Assert
	require.ErrorIs(t, err, publishErr)
	require.ErrorIs(t, err, deleteAvatarErr)
	require.ErrorIs(t, err, deleteObjectErr)
	assert.Zero(t, result)
	require.Len(t, avatarRepo.created, 1)
	assert.Equal(t, []uuid.UUID{avatarRepo.created[0].ID}, avatarRepo.deletedIDs)
	assert.Equal(t, []string{testObjectKeyOriginal}, fileStore.deletes)
}
