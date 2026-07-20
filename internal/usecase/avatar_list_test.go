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

// TestAvatarUseCase_ListUserAvatars проверяет получение списка неудаленных аватарок пользователя.
func TestAvatarUseCase_ListUserAvatars(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	processingAvatarID := uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a005")
	deletingAvatarID := uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a006")
	foreignAvatarID := uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a007")
	currentAvatar := mustReadyListAvatar(t, testAvatarID, now)
	processingAvatar := mustProcessingListAvatar(t, processingAvatarID, now)
	deletingAvatar := mustReadyListAvatar(t, deletingAvatarID, now)
	require.NoError(t, deletingAvatar.MarkDeleting(now))
	foreignAvatar := mustReadyListAvatar(t, foreignAvatarID, now)
	foreignAvatar.UserID = testOtherUserID
	user := mustUseCaseUser(t, now)
	require.NoError(t, user.SelectCurrentAvatar(currentAvatar, now))
	userRepo := &avatarUserRepositoryFake{user: user}
	avatarRepo := &avatarRepositoryFake{
		avatars: []model.Avatar{
			currentAvatar,
			processingAvatar,
			deletingAvatar,
			foreignAvatar,
		},
	}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	output, err := useCase.ListUserAvatars(ctx, ListUserAvatarsInput{UserID: testUserID})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Equal(t, []uuid.UUID{testUserID}, avatarRepo.userIDs)
	require.Len(t, output.Avatars, 2)
	assert.Equal(t, testAvatarID, output.Avatars[0].ID)
	assert.True(t, output.Avatars[0].IsCurrent)
	assert.Equal(t, processingAvatarID, output.Avatars[1].ID)
	assert.False(t, output.Avatars[1].IsCurrent)
}

// TestAvatarUseCase_ListUserAvatars_ReturnsUserNotFound проверяет ошибку отсутствия пользователя.
func TestAvatarUseCase_ListUserAvatars_ReturnsUserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo := &avatarUserRepositoryFake{err: ErrUserNotFound}
	avatarRepo := &avatarRepositoryFake{}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	output, err := useCase.ListUserAvatars(ctx, ListUserAvatarsInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Zero(t, output)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Empty(t, avatarRepo.userIDs)
}

// TestAvatarUseCase_ListUserAvatars_ReturnsListError проверяет ошибку чтения списка аватарок.
func TestAvatarUseCase_ListUserAvatars_ReturnsListError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	listErr := errors.New("list avatars error")
	userRepo := &avatarUserRepositoryFake{user: mustUseCaseUser(t, now)}
	avatarRepo := &avatarRepositoryFake{listErr: listErr}
	useCase := mustAvatarUseCase(t, userRepo, avatarRepo, &fileStoreFake{}, &avatarMessagePublisherFake{})

	// Act
	output, err := useCase.ListUserAvatars(ctx, ListUserAvatarsInput{UserID: testUserID})

	// Assert
	require.ErrorIs(t, err, listErr)
	assert.Zero(t, output)
	assert.Equal(t, []uuid.UUID{testUserID}, userRepo.ids)
	assert.Equal(t, []uuid.UUID{testUserID}, avatarRepo.userIDs)
}

func mustReadyListAvatar(t *testing.T, avatarID uuid.UUID, now time.Time) model.Avatar {
	t.Helper()

	avatar := mustProcessingListAvatar(t, avatarID, now)
	err := avatar.MarkReady(
		100,
		100,
		"thumbs/"+avatarID.String()+"/100.png",
		"thumbs/"+avatarID.String()+"/300.png",
		now,
	)
	require.NoError(t, err)

	return avatar
}

func mustProcessingListAvatar(t *testing.T, avatarID uuid.UUID, now time.Time) model.Avatar {
	t.Helper()

	avatar, err := model.NewProcessingAvatar(
		avatarID,
		testUserID,
		"avatar.png",
		model.MIMEPNG,
		100,
		"users/user-id/avatars/"+avatarID.String()+"/original",
		now,
	)
	require.NoError(t, err)

	return avatar
}
