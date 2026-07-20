package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// TestAvatarUseCase_GetAvatar проверяет получение готовой аватарки нужного размера.
func TestAvatarUseCase_GetAvatar(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	avatar := mustReadyUseCaseAvatar(t, now)

	tests := []struct {
		name          string
		size          AvatarSize
		wantObjectKey string
		wantMIMEType  string
	}{
		{
			name:          "original",
			size:          AvatarSizeOriginal,
			wantObjectKey: testObjectKeyOriginal,
			wantMIMEType:  model.MIMEPNG,
		},
		{
			name:          "100x100 thumbnail",
			size:          AvatarSize100,
			wantObjectKey: *avatar.ObjectKeyThumb100,
			wantMIMEType:  model.MIMEPNG,
		},
		{
			name:          "300x300 thumbnail",
			size:          AvatarSize300,
			wantObjectKey: *avatar.ObjectKeyThumb300,
			wantMIMEType:  model.MIMEPNG,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			fileStore := &fileStoreFake{content: []byte("avatar content")}
			useCase := mustAvatarUseCase(
				t,
				&avatarUserRepositoryFake{},
				&avatarRepositoryFake{avatar: avatar},
				fileStore,
				&avatarMessagePublisherFake{},
			)

			// Act
			result, err := useCase.GetAvatar(ctx, GetAvatarInput{
				AvatarID: testAvatarID,
				Size:     tt.size,
				MIMEType: tt.wantMIMEType,
			})

			// Assert
			require.NoError(t, err)
			assert.Equal(t, []byte("avatar content"), result.Content)
			assert.Equal(t, tt.wantMIMEType, result.MIMEType)
			assert.Equal(t, []string{tt.wantObjectKey}, fileStore.gets)
		})
	}
}

// TestAvatarUseCase_GetAvatar_ReturnsAvatarNotFound проверяет случаи, когда файл аватарки нельзя выдать.
func TestAvatarUseCase_GetAvatar_ReturnsAvatarNotFound(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	processingAvatar := mustProcessingUseCaseAvatar(t, now)
	readyAvatar := mustReadyUseCaseAvatar(t, now)
	readyAvatar.ObjectKeyThumb100 = nil
	deletedAvatar := mustReadyUseCaseAvatar(t, now)
	require.NoError(t, deletedAvatar.MarkDeleting(now))

	tests := []struct {
		name      string
		avatar    model.Avatar
		avatarErr error
		size      AvatarSize
	}{
		{
			name:      "avatar record is missing",
			avatar:    model.Avatar{},
			avatarErr: ErrAvatarNotFound,
			size:      AvatarSizeOriginal,
		},
		{
			name:      "avatar is not ready",
			avatar:    processingAvatar,
			avatarErr: nil,
			size:      AvatarSizeOriginal,
		},
		{
			name:      "avatar is deleted",
			avatar:    deletedAvatar,
			avatarErr: nil,
			size:      AvatarSizeOriginal,
		},
		{
			name:      "thumbnail is missing",
			avatar:    readyAvatar,
			avatarErr: nil,
			size:      AvatarSize100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			fileStore := &fileStoreFake{content: []byte("avatar content")}
			useCase := mustAvatarUseCase(
				t,
				&avatarUserRepositoryFake{},
				&avatarRepositoryFake{avatar: tt.avatar, getErr: tt.avatarErr},
				fileStore,
				&avatarMessagePublisherFake{},
			)

			// Act
			result, err := useCase.GetAvatar(ctx, GetAvatarInput{
				AvatarID: testAvatarID,
				Size:     tt.size,
			})

			// Assert
			require.ErrorIs(t, err, ErrAvatarNotFound)
			assert.Zero(t, result)
			assert.Empty(t, fileStore.gets)
		})
	}
}

// TestAvatarUseCase_GetAvatar_RejectsFormatMismatch проверяет ошибку несовпадения запрошенного формата.
func TestAvatarUseCase_GetAvatar_RejectsFormatMismatch(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	fileStore := &fileStoreFake{content: []byte("avatar content")}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		&avatarRepositoryFake{avatar: mustReadyUseCaseAvatar(t, now)},
		fileStore,
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.GetAvatar(ctx, GetAvatarInput{
		AvatarID: testAvatarID,
		Size:     AvatarSizeOriginal,
		MIMEType: model.MIMEJPEG,
	})

	// Assert
	require.ErrorIs(t, err, model.ErrInvalidAvatarMetadata)
	assert.Zero(t, result)
	assert.Empty(t, fileStore.gets)
}

// TestAvatarUseCase_GetAvatar_ReturnsStorageError проверяет ошибку чтения файла аватарки.
func TestAvatarUseCase_GetAvatar_ReturnsStorageError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	storageErr := errors.New("storage error")
	fileStore := &fileStoreFake{getErr: storageErr}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{},
		&avatarRepositoryFake{avatar: mustReadyUseCaseAvatar(t, now)},
		fileStore,
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.GetAvatar(ctx, GetAvatarInput{
		AvatarID: testAvatarID,
		Size:     AvatarSizeOriginal,
	})

	// Assert
	require.ErrorIs(t, err, storageErr)
	assert.Zero(t, result)
	assert.Equal(t, []string{testObjectKeyOriginal}, fileStore.gets)
}
