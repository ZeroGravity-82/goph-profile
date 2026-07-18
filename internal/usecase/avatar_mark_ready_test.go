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

// TestAvatarUseCase_MarkAvatarReady проверяет успешное завершение обработки аватарки.
func TestAvatarUseCase_MarkAvatarReady(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := validMarkAvatarReadyInput()

	// Act
	result, err := useCase.MarkAvatarReady(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testAvatarID, result.ID)
	assert.Equal(t, testUserID, result.UserID)
	assert.Equal(t, model.AvatarStatusReady, result.Status)
	assert.Equal(t, input.Width, result.Width)
	assert.Equal(t, input.Height, result.Height)
	assert.Equal(t, input.ObjectKeyThumb100, result.ObjectKeyThumb100)
	assert.Equal(t, input.ObjectKeyThumb300, result.ObjectKeyThumb300)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	require.Len(t, avatarRepo.updated, 1)
	assert.Equal(t, model.AvatarStatusReady, avatarRepo.updated[0].Status)
	require.Len(t, userRepo.updated, 1)
	require.NotNil(t, userRepo.updated[0].CurrentAvatarID)
	assert.Equal(t, testAvatarID, *userRepo.updated[0].CurrentAvatarID)
}

// TestAvatarUseCase_MarkAvatarReady_DoesNotReplaceCurrentAvatar проверяет сохранение уже выбранной текущей аватарки
// при завершении обработки.
func TestAvatarUseCase_MarkAvatarReady_DoesNotReplaceCurrentAvatar(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	user := mustUseCaseUser(t, now)
	currentAvatarID := testOtherAvatarID
	user.CurrentAvatarID = &currentAvatarID
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	result, err := useCase.MarkAvatarReady(ctx, validMarkAvatarReadyInput())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.AvatarStatusReady, result.Status)
	require.Len(t, avatarRepo.updated, 1)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_MarkAvatarReady_ReturnsAvatarNotFound проверяет ошибку отсутствия аватарки при завершении
// обработки.
func TestAvatarUseCase_MarkAvatarReady_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo := &avatarUserRepositoryFake{}
	avatarRepo := &avatarRepositoryFake{getErr: ErrAvatarNotFound}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	result, err := useCase.MarkAvatarReady(ctx, validMarkAvatarReadyInput())

	// Assert
	require.ErrorIs(t, err, ErrAvatarNotFound)
	assert.Zero(t, result)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	assert.Empty(t, userRepo.ids)
	assert.Empty(t, avatarRepo.updated)
}

// TestAvatarUseCase_MarkAvatarReady_ReturnsInvalidMetadata проверяет ошибку невалидных данных при завершении
// обработки.
func TestAvatarUseCase_MarkAvatarReady_ReturnsInvalidMetadata(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	userRepo := &avatarUserRepositoryFake{}
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})
	input := validMarkAvatarReadyInput()
	input.Width = 0

	// Act
	result, err := useCase.MarkAvatarReady(ctx, input)

	// Assert
	require.ErrorIs(t, err, model.ErrInvalidAvatarMetadata)
	assert.Zero(t, result)
	assert.Empty(t, userRepo.ids)
	assert.Empty(t, avatarRepo.updated)
}

// TestAvatarUseCase_MarkAvatarReady_ReturnsUserNotFound проверяет ошибку отсутствия пользователя при завершении
// обработки аватарки.
func TestAvatarUseCase_MarkAvatarReady_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	userRepo := &avatarUserRepositoryFake{err: ErrUserNotFound}
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	result, err := useCase.MarkAvatarReady(ctx, validMarkAvatarReadyInput())

	// Assert
	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Zero(t, result)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Empty(t, avatarRepo.updated)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_MarkAvatarReady_ReturnsUpdateAvatarError проверяет ошибку сохранения готовой аватарки при
// завершении обработки.
func TestAvatarUseCase_MarkAvatarReady_ReturnsUpdateAvatarError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updateErr := errors.New("update avatar error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now), updateErr: updateErr}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	result, err := useCase.MarkAvatarReady(ctx, validMarkAvatarReadyInput())

	// Assert
	require.ErrorIs(t, err, updateErr)
	assert.Zero(t, result)
	require.Len(t, avatarRepo.updated, 1)
	assert.Empty(t, userRepo.updated)
}

// TestAvatarUseCase_MarkAvatarReady_ReturnsUpdateUserError проверяет ошибку выбора первой текущей аватарки при
// завершении обработки.
func TestAvatarUseCase_MarkAvatarReady_ReturnsUpdateUserError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updateErr := errors.New("update user error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now), updateErr: updateErr}
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	result, err := useCase.MarkAvatarReady(ctx, validMarkAvatarReadyInput())

	// Assert
	require.ErrorIs(t, err, updateErr)
	assert.Zero(t, result)
	require.Len(t, avatarRepo.updated, 1)
	require.Len(t, userRepo.updated, 1)
}

func validMarkAvatarReadyInput() MarkAvatarReadyInput {
	return MarkAvatarReadyInput{
		AvatarID:          testAvatarID,
		Width:             100,
		Height:            100,
		ObjectKeyThumb100: "thumbs/avatar-id/100.png",
		ObjectKeyThumb300: "thumbs/avatar-id/300.png",
	}
}
