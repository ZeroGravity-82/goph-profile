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
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// TestAvatarUseCase_SelectCurrentAvatar проверяет успешный выбор текущей аватарки.
func TestAvatarUseCase_SelectCurrentAvatar(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	user := mustUseCaseUser(t, now)
	avatar := mustReadyUseCaseAvatar(t, now)
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := SelectCurrentAvatarInput{UserID: testUserID, AvatarID: testAvatarID}

	// Act
	err := useCase.SelectCurrentAvatar(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.lockIDs)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	require.Len(t, userRepo.updated, 1)
	require.NotNil(t, userRepo.updated[0].CurrentAvatarID)
	assert.Equal(t, testAvatarID, *userRepo.updated[0].CurrentAvatarID)
}

// TestAvatarUseCase_SelectCurrentAvatar_IsIdempotent проверяет повторный выбор текущей аватарки.
func TestAvatarUseCase_SelectCurrentAvatar_IsIdempotent(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	user := mustUseCaseUser(t, now)
	avatar := mustReadyUseCaseAvatar(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := SelectCurrentAvatarInput{UserID: testUserID, AvatarID: testAvatarID}

	// Act
	err := useCase.SelectCurrentAvatar(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.lockIDs)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_SelectCurrentAvatar_ReturnsUserNotFound проверяет ошибку отсутствия пользователя при выборе
// текущей аватарки.
func TestAvatarUseCase_SelectCurrentAvatar_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo := &avatarUserRepositoryFake{err: repository.ErrUserNotFound}
	avatarRepo := &avatarRepositoryFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := SelectCurrentAvatarInput{UserID: testUserID, AvatarID: testAvatarID}

	// Act
	err := useCase.SelectCurrentAvatar(ctx, input)

	// Assert
	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Empty(t, avatarRepo.ids)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_SelectCurrentAvatar_ReturnsAvatarNotFound проверяет ошибку отсутствия аватарки при выборе
// текущей аватарки.
func TestAvatarUseCase_SelectCurrentAvatar_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{getErr: repository.ErrAvatarNotFound}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := SelectCurrentAvatarInput{UserID: testUserID, AvatarID: testAvatarID}

	// Act
	err := useCase.SelectCurrentAvatar(ctx, input)

	// Assert
	require.ErrorIs(t, err, repository.ErrAvatarNotFound)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_SelectCurrentAvatar_ReturnsAvatarForbidden проверяет ошибку выбора чужой аватарки в качестве
// текущей.
func TestAvatarUseCase_SelectCurrentAvatar_ReturnsAvatarForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatar := mustReadyUseCaseAvatar(t, now)
	avatar.UserID = testOtherUserID
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := SelectCurrentAvatarInput{UserID: testUserID, AvatarID: testAvatarID}

	// Act
	err := useCase.SelectCurrentAvatar(ctx, input)

	// Assert
	require.ErrorIs(t, err, model.ErrAvatarForbidden)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_SelectCurrentAvatar_ReturnsAvatarNotReady проверяет ошибку выбора неготовой аватарки в качестве
// текущей.
func TestAvatarUseCase_SelectCurrentAvatar_ReturnsAvatarNotReady(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := SelectCurrentAvatarInput{UserID: testUserID, AvatarID: testAvatarID}

	// Act
	err := useCase.SelectCurrentAvatar(ctx, input)

	// Assert
	require.ErrorIs(t, err, model.ErrAvatarNotReady)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_SelectCurrentAvatar_ReturnsUpdateUserError проверяет ошибку сохранения пользователя при выборе
// текущей аватарки.
func TestAvatarUseCase_SelectCurrentAvatar_ReturnsUpdateUserError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updateErr := errors.New("update user error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now), updateErr: updateErr}
	avatarRepo := &avatarRepositoryFake{avatar: mustReadyUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := SelectCurrentAvatarInput{UserID: testUserID, AvatarID: testAvatarID}

	// Act
	err := useCase.SelectCurrentAvatar(ctx, input)

	// Assert
	require.ErrorIs(t, err, updateErr)
	require.Len(t, userRepo.updated, 1)
	require.NotNil(t, userRepo.updated[0].CurrentAvatarID)
	assert.Equal(t, testAvatarID, *userRepo.updated[0].CurrentAvatarID)
}
