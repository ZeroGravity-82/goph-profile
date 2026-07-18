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

// TestAvatarUseCase_MarkAvatarFailed проверяет завершение обработки аватарки ошибкой.
func TestAvatarUseCase_MarkAvatarFailed(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
		&fileStoreFake{},
		&avatarMessagePublisherFake{},
	)
	input := MarkAvatarFailedInput{AvatarID: testAvatarID}

	// Act
	result, err := useCase.MarkAvatarFailed(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testAvatarID, result.ID)
	assert.Equal(t, testUserID, result.UserID)
	assert.Equal(t, model.AvatarStatusFailed, result.Status)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	require.Len(t, avatarRepo.updated, 1)
	assert.Equal(t, model.AvatarStatusFailed, avatarRepo.updated[0].Status)
}

// TestAvatarUseCase_MarkAvatarFailed_ReturnsAvatarNotFound проверяет ошибку отсутствия аватарки.
func TestAvatarUseCase_MarkAvatarFailed_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	avatarRepo := &avatarRepositoryFake{getErr: ErrAvatarNotFound}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
		&fileStoreFake{},
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.MarkAvatarFailed(ctx, MarkAvatarFailedInput{AvatarID: testAvatarID})

	// Assert
	require.ErrorIs(t, err, ErrAvatarNotFound)
	assert.Zero(t, result)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	assert.Empty(t, avatarRepo.updated)
}

// TestAvatarUseCase_MarkAvatarFailed_ReturnsInvalidTransition проверяет ошибку недопустимого перехода статуса.
func TestAvatarUseCase_MarkAvatarFailed_ReturnsInvalidTransition(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatarRepo := &avatarRepositoryFake{avatar: mustReadyUseCaseAvatar(t, now)}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
		&fileStoreFake{},
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.MarkAvatarFailed(ctx, MarkAvatarFailedInput{AvatarID: testAvatarID})

	// Assert
	require.ErrorIs(t, err, model.ErrInvalidAvatarTransition)
	assert.Zero(t, result)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	assert.Empty(t, avatarRepo.updated)
}

// TestAvatarUseCase_MarkAvatarFailed_ReturnsUpdateAvatarError проверяет ошибку сохранения аватарки.
func TestAvatarUseCase_MarkAvatarFailed_ReturnsUpdateAvatarError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	updateErr := errors.New("update avatar error")
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now), updateErr: updateErr}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
		&fileStoreFake{},
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.MarkAvatarFailed(ctx, MarkAvatarFailedInput{AvatarID: testAvatarID})

	// Assert
	require.ErrorIs(t, err, updateErr)
	assert.Zero(t, result)
	require.Len(t, avatarRepo.updated, 1)
	assert.Equal(t, model.AvatarStatusFailed, avatarRepo.updated[0].Status)
}
