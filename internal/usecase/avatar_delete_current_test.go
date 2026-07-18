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

// TestAvatarUseCase_DeleteCurrentAvatar проверяет удаление текущей аватарки.
func TestAvatarUseCase_DeleteCurrentAvatar(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatar := mustReadyUseCaseAvatar(t, now)
	user := mustUseCaseUser(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, messagePublisher)

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	require.Len(t, avatarRepo.updated, 1)
	assert.Equal(t, model.AvatarStatusDeleting, avatarRepo.updated[0].Status)
	require.NotNil(t, avatarRepo.updated[0].DeletedAt)
	require.Len(t, userRepo.updated, 1)
	assert.Nil(t, userRepo.updated[0].CurrentAvatarID)
	assert.Equal(t, []AvatarDeletionMessage{
		{
			AvatarID: testAvatarID,
			ObjectKeys: []string{
				testObjectKeyOriginal,
				"thumbs/avatar-id/100.png",
				"thumbs/avatar-id/300.png",
			},
		},
	}, messagePublisher.deletionMessages)
}

// TestAvatarUseCase_DeleteCurrentAvatar_IsIdempotent проверяет удаление при отсутствии текущей аватарки.
func TestAvatarUseCase_DeleteCurrentAvatar_IsIdempotent(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, messagePublisher)

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Empty(t, avatarRepo.ids)
	assert.Empty(t, avatarRepo.updated)
	assert.Empty(t, userRepo.updated)
	assert.Empty(t, messagePublisher.deletionMessages)
}

// TestAvatarUseCase_DeleteCurrentAvatar_ReturnsUserNotFound проверяет ошибку отсутствия пользователя при удалении
// аватарки.
func TestAvatarUseCase_DeleteCurrentAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo := &avatarUserRepositoryFake{err: ErrUserNotFound}
	avatarRepo := &avatarRepositoryFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Empty(t, avatarRepo.ids)
}

// TestAvatarUseCase_DeleteCurrentAvatar_ReturnsAvatarNotFound проверяет ошибку отсутствия текущей аватарки при
// удалении.
func TestAvatarUseCase_DeleteCurrentAvatar_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatar := mustReadyUseCaseAvatar(t, now)
	user := mustUseCaseUser(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{getErr: ErrAvatarNotFound}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, ErrAvatarNotFound)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	assert.Empty(t, avatarRepo.updated)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_DeleteCurrentAvatar_ReturnsAvatarForbidden проверяет ошибку удаления чужой аватарки.
func TestAvatarUseCase_DeleteCurrentAvatar_ReturnsAvatarForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatar := mustReadyUseCaseAvatar(t, now)
	user := mustUseCaseUser(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))
	avatar.UserID = testOtherUserID
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, model.ErrAvatarForbidden)
	assert.Empty(t, avatarRepo.updated)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_DeleteCurrentAvatar_ReturnsUpdateAvatarError проверяет ошибку сохранения аватарки при удалении.
func TestAvatarUseCase_DeleteCurrentAvatar_ReturnsUpdateAvatarError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updateErr := errors.New("update avatar error")
	avatar := mustReadyUseCaseAvatar(t, now)
	user := mustUseCaseUser(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: avatar, updateErr: updateErr}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, messagePublisher)

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, updateErr)
	require.Len(t, avatarRepo.updated, 1)
	assert.Empty(t, userRepo.updated)
	assert.Empty(t, messagePublisher.deletionMessages)
}

// TestAvatarUseCase_DeleteCurrentAvatar_ReturnsUpdateUserError проверяет ошибку сохранения пользователя при удалении
// аватарки.
func TestAvatarUseCase_DeleteCurrentAvatar_ReturnsUpdateUserError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updateErr := errors.New("update user error")
	avatar := mustReadyUseCaseAvatar(t, now)
	user := mustUseCaseUser(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))
	userRepo := &avatarUserRepositoryFake{user: user, updateErr: updateErr}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	messagePublisher := &avatarMessagePublisherFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, messagePublisher)

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, updateErr)
	require.Len(t, avatarRepo.updated, 1)
	require.Len(t, userRepo.updated, 1)
	assert.Empty(t, messagePublisher.deletionMessages)
}

// TestAvatarUseCase_DeleteCurrentAvatar_ReturnsPublishDeletionError проверяет ошибку публикации сообщения об удалении
// аватарки.
func TestAvatarUseCase_DeleteCurrentAvatar_ReturnsPublishDeletionError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	publishErr := errors.New("publish deletion error")
	avatar := mustReadyUseCaseAvatar(t, now)
	user := mustUseCaseUser(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	messagePublisher := &avatarMessagePublisherFake{deleteErr: publishErr}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, messagePublisher)

	// Act
	err := useCase.DeleteCurrentAvatar(ctx, DeleteCurrentAvatarInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, publishErr)
	require.Len(t, avatarRepo.updated, 1)
	require.Len(t, userRepo.updated, 1)
	require.Len(t, messagePublisher.deletionMessages, 1)
}
