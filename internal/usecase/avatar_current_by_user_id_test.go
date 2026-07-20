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

// TestAvatarUseCase_GetCurrentAvatarByUserID проверяет получение текущей готовой аватарки по ID пользователя.
func TestAvatarUseCase_GetCurrentAvatarByUserID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	user := mustUseCaseUser(t, now)
	avatar := mustReadyUseCaseAvatar(t, now)
	user.CurrentAvatarID = &avatar.ID
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{avatar: avatar}
	fileStore := &fileStoreFake{content: []byte("avatar content")}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, &avatarMessagePublisherFake{})

	// Act
	result, err := useCase.GetCurrentAvatarByUserID(ctx, GetCurrentAvatarByUserIDInput{UserID: testUserID})

	// Assert
	require.NoError(t, err)
	assert.False(t, result.UseDefaultAvatar)
	assert.Equal(t, []byte("avatar content"), result.Content)
	assert.Equal(t, model.MIMEPNG, result.MIMEType)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Equal(t, []string{testObjectKeyOriginal}, fileStore.gets)
	assert.Empty(t, fileStore.deletes)
}

// TestAvatarUseCase_GetCurrentAvatarByUserID_ReturnsDefaultAvatar проверяет выдачу заглушки.
func TestAvatarUseCase_GetCurrentAvatarByUserID_ReturnsDefaultAvatar(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	readyAvatar := mustReadyUseCaseAvatar(t, now)
	currentUser := mustUseCaseUser(t, now)
	currentUser.CurrentAvatarID = &readyAvatar.ID
	processingAvatar := mustProcessingUseCaseAvatar(t, now)

	tests := []struct {
		name             string
		user             model.User
		userErr          error
		avatar           model.Avatar
		avatarErr        error
		wantAvatarLookup bool
	}{
		{
			name:             "returns default avatar when user does not exist",
			user:             model.User{},
			userErr:          ErrUserNotFound,
			avatar:           model.Avatar{},
			avatarErr:        nil,
			wantAvatarLookup: false,
		},
		{
			name:             "returns default avatar when user has no selected current avatar",
			user:             mustUseCaseUser(t, now),
			userErr:          nil,
			avatar:           model.Avatar{},
			avatarErr:        nil,
			wantAvatarLookup: false,
		},
		{
			name:             "returns default avatar when selected current avatar record is missing",
			user:             currentUser,
			userErr:          nil,
			avatar:           model.Avatar{},
			avatarErr:        ErrAvatarNotFound,
			wantAvatarLookup: true,
		},
		{
			name:             "returns default avatar when selected current avatar is still processing",
			user:             currentUser,
			userErr:          nil,
			avatar:           processingAvatar,
			avatarErr:        nil,
			wantAvatarLookup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			userRepo := &avatarUserRepositoryFake{user: tt.user, err: tt.userErr}
			avatarRepo := &avatarRepositoryFake{avatar: tt.avatar, getErr: tt.avatarErr}
			fileStore := &fileStoreFake{content: []byte("avatar content")}
			useCase := mustAvatarUseCase(t, userRepo, avatarRepo, fileStore, &avatarMessagePublisherFake{})

			// Act
			result, err := useCase.GetCurrentAvatarByUserID(ctx, GetCurrentAvatarByUserIDInput{UserID: testUserID})

			// Assert
			require.NoError(t, err)
			assert.True(t, result.UseDefaultAvatar)
			assert.Empty(t, result.Content)
			assert.Empty(t, result.MIMEType)
			assert.Empty(t, fileStore.gets)
			assert.Equal(t, tt.wantAvatarLookup, len(avatarRepo.ids) > 0)
		})
	}
}

// TestAvatarUseCase_GetCurrentAvatarByUserID_ReturnsAvatarForbidden проверяет ошибку ссылки на чужую аватарку.
func TestAvatarUseCase_GetCurrentAvatarByUserID_ReturnsAvatarForbidden(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	user := mustUseCaseUser(t, now)
	avatar := mustReadyUseCaseAvatar(t, now)
	user.CurrentAvatarID = &avatar.ID
	avatar.UserID = testOtherUserID
	fileStore := &fileStoreFake{content: []byte("avatar content")}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{user: user},
		&avatarRepositoryFake{avatar: avatar},
		fileStore,
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.GetCurrentAvatarByUserID(ctx, GetCurrentAvatarByUserIDInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, model.ErrAvatarForbidden)
	assert.Zero(t, result)
	assert.Empty(t, fileStore.gets)
}

// TestAvatarUseCase_GetCurrentAvatarByUserID_ReturnsStorageError проверяет ошибку чтения файла аватарки.
func TestAvatarUseCase_GetCurrentAvatarByUserID_ReturnsStorageError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	user := mustUseCaseUser(t, now)
	avatar := mustReadyUseCaseAvatar(t, now)
	user.CurrentAvatarID = &avatar.ID
	storageErr := errors.New("storage error")
	fileStore := &fileStoreFake{getErr: storageErr}
	useCase := mustAvatarUseCase(
		t,
		&avatarUserRepositoryFake{user: user},
		&avatarRepositoryFake{avatar: avatar},
		fileStore,
		&avatarMessagePublisherFake{},
	)

	// Act
	result, err := useCase.GetCurrentAvatarByUserID(ctx, GetCurrentAvatarByUserIDInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, storageErr)
	assert.Zero(t, result)
	assert.Equal(t, []string{testObjectKeyOriginal}, fileStore.gets)
}
