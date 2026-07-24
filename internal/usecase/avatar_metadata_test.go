package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// TestAvatarUseCase_GetAvatarMetadata проверяет получение метаданных аватарки.
func TestAvatarUseCase_GetAvatarMetadata(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatar := mustReadyUseCaseAvatar(t, now)
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
		&fileStoreFake{},
		&avatarMessagePublisherFake{},
	)
	input := GetAvatarMetadataInput{AvatarID: testAvatarID}

	// Act
	result, err := useCase.GetAvatarMetadata(ctx, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testAvatarID, result.ID)
	assert.Equal(t, testUserID, result.UserID)
	assert.Equal(t, "avatar.png", result.FileName)
	assert.Equal(t, model.MIMEPNG, result.MIMEType)
	assert.Equal(t, int64(100), result.SizeBytes)
	assert.Equal(t, model.AvatarStatusReady, result.Status)
	assert.Equal(t, now, result.CreatedAt)
	assert.Equal(t, now, result.UpdatedAt)
	require.NotNil(t, result.Width)
	assert.Equal(t, 100, *result.Width)
	require.NotNil(t, result.Height)
	assert.Equal(t, 100, *result.Height)
	assert.Equal(t, testObjectKeyOriginal, result.ObjectKeyOriginal)
	require.NotNil(t, result.ObjectKeyThumb100)
	assert.Equal(t, "thumbs/avatar-id/100.png", *result.ObjectKeyThumb100)
	require.NotNil(t, result.ObjectKeyThumb300)
	assert.Equal(t, "thumbs/avatar-id/300.png", *result.ObjectKeyThumb300)
	assert.Nil(t, result.DeletedAt)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
}

// TestAvatarUseCase_GetAvatarMetadata_ReturnsAvatarNotFound проверяет ошибку отсутствия аватарки.
func TestAvatarUseCase_GetAvatarMetadata_ReturnsAvatarNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	avatarRepo := &avatarRepositoryFake{getErr: repository.ErrAvatarNotFound}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		avatarRepo,
		&fileStoreFake{},
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.GetAvatarMetadata(ctx, GetAvatarMetadataInput{AvatarID: testAvatarID})

	// Assert
	require.ErrorIs(t, err, repository.ErrAvatarNotFound)
	assert.Zero(t, result)
	assert.Equal(t, []uuid.UUID{testAvatarID}, avatarRepo.ids)
}
