package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// TestAvatarWorkerUseCase_MarkAvatarDeleted проверяет завершение удаления файлов аватарки.
func TestAvatarWorkerUseCase_MarkAvatarDeleted(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatar := mustReadyUseCaseAvatar(t, now)
	require.NoError(t, avatar.MarkDeleting(now))
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	useCase := mustAvatarWorkerUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
	)

	// Act
	err := useCase.MarkAvatarDeleted(ctx, MarkAvatarDeletedInput{AvatarID: testAvatarID})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
	require.Len(t, avatarRepo.updated, 1)
	assert.Equal(t, model.AvatarStatusDeleted, avatarRepo.updated[0].Status)
}

// TestAvatarWorkerUseCase_MarkAvatarDeleted_ReturnsInvalidTransition проверяет ошибку недопустимого перехода статуса.
func TestAvatarWorkerUseCase_MarkAvatarDeleted_ReturnsInvalidTransition(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatarRepo := &avatarRepositoryFake{avatar: mustProcessingUseCaseAvatar(t, now)}
	useCase := mustAvatarWorkerUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
	)

	// Act
	err := useCase.MarkAvatarDeleted(ctx, MarkAvatarDeletedInput{AvatarID: testAvatarID})

	// Assert
	require.ErrorIs(t, err, model.ErrInvalidAvatarTransition)
	assert.Empty(t, avatarRepo.updated)
}
