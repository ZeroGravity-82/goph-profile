package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

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

func mustReadyUseCaseAvatar(t *testing.T, now time.Time) model.Avatar {
	t.Helper()

	avatar := mustProcessingUseCaseAvatar(t, now)
	err := avatar.MarkReady(
		100,
		100,
		"thumbs/avatar-id/100.png",
		"thumbs/avatar-id/300.png",
		now,
	)
	require.NoError(t, err)

	return avatar
}

func mustProcessingUseCaseAvatar(t *testing.T, now time.Time) model.Avatar {
	t.Helper()

	avatar, err := model.NewProcessingAvatar(
		testAvatarID,
		testUserID,
		"avatar.png",
		model.MIMEPNG,
		100,
		testObjectKeyOriginal,
		now,
	)
	require.NoError(t, err)

	return avatar
}
