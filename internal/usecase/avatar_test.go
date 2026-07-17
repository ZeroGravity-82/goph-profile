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

const testObjectKeyOriginal = "users/user-id/avatars/avatar-id/original"

// TestNewAvatarUseCase_RejectsNilDependencies проверяет обязательность зависимостей.
func TestNewAvatarUseCase_RejectsNilDependencies(t *testing.T) {
	validUserRepo := &avatarUserRepositoryFake{}
	validAvatarRepo := &avatarRepositoryFake{}
	validFileStore := &fileStoreFake{}
	validMessagePublisher := &avatarMessagePublisherFake{}

	tests := []struct {
		name             string
		userRepo         avatarUserRepository
		avatarRepo       avatarRepository
		fileStore        fileStorage
		messagePublisher avatarMessagePublisher
		wantErr          string
	}{
		{
			name:             "user repository",
			avatarRepo:       validAvatarRepo,
			fileStore:        validFileStore,
			messagePublisher: validMessagePublisher,
			wantErr:          "user repository is not provided",
		},
		{
			name:             "avatar repository",
			userRepo:         validUserRepo,
			fileStore:        validFileStore,
			messagePublisher: validMessagePublisher,
			wantErr:          "avatar repository is not provided",
		},
		{
			name:             "file storage",
			userRepo:         validUserRepo,
			avatarRepo:       validAvatarRepo,
			messagePublisher: validMessagePublisher,
			wantErr:          "file storage is not provided",
		},
		{
			name:       "message publisher",
			userRepo:   validUserRepo,
			avatarRepo: validAvatarRepo,
			fileStore:  validFileStore,
			wantErr:    "avatar message publisher is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			useCase, err := NewAvatarUseCase(
				tt.userRepo,
				tt.avatarRepo,
				tt.fileStore,
				tt.messagePublisher,
			)

			// Assert
			require.EqualError(t, err, tt.wantErr)
			assert.Nil(t, useCase)
		})
	}
}

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

// TestAvatarUseCase_UploadAvatar_RejectsInvalidMetadata проверяет ошибку при невалидных метаданных аватарки.
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

// TestAvatarUseCase_UploadAvatar_ReturnsUserNotFound проверяет ошибку отсутствия пользователя.
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

// TestAvatarUseCase_UploadAvatar_ReturnsPutOriginalObjectError проверяет ошибку сохранения исходного файла.
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

// TestAvatarUseCase_UploadAvatar_DeletesOriginalObjectAfterCreateError проверяет удаление исходного файла
// из хранилища после ошибки создания записи аватарки в БД.
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

// TestAvatarUseCase_UploadAvatar_DeletesAvatarAndOriginalObjectAfterPublishError проверяет удаление записи
// аватарки в БД и исходного файла после ошибки публикации сообщения.
func TestAvatarUseCase_UploadAvatar_DeletesAvatarAndOriginalObjectAfterPublishError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	publishErr := errors.New("publish error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{}
	fileStore := &fileStoreFake{objectKey: testObjectKeyOriginal}
	messagePublisher := &avatarMessagePublisherFake{err: publishErr}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, messagePublisher)

	// Act
	result, err := useCase.UploadAvatar(ctx, validUploadAvatarInput())

	// Assert
	require.ErrorIs(t, err, publishErr)
	assert.Zero(t, result)
	require.Len(t, avatarRepo.created, 1)
	assert.Equal(t, []uuid.UUID{avatarRepo.created[0].ID}, avatarRepo.deletedIDs)
	assert.Equal(t, []string{testObjectKeyOriginal}, fileStore.deletes)
}

func validUploadAvatarInput() UploadAvatarInput {
	return UploadAvatarInput{
		UserID:   testUserID,
		FileName: "avatar.png",
		MIMEType: model.MIMEPNG,
		Content:  []byte("image content"),
	}
}

func mustAvatarUseCase(
	t *testing.T,
	userRepo avatarUserRepository,
	avatarRepo avatarRepository,
	fileStore fileStorage,
	messagePublisher avatarMessagePublisher,
) *AvatarUseCase {
	t.Helper()

	useCase, err := NewAvatarUseCase(userRepo, avatarRepo, fileStore, messagePublisher)
	require.NoError(t, err)

	return useCase
}

func mustUseCaseUser(t *testing.T, now time.Time) model.User {
	t.Helper()

	user, err := model.NewUser(testUserID, "user@example.com", now)
	require.NoError(t, err)

	return user
}
