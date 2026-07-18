package usecase

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

const testObjectKeyOriginal = "users/user-id/avatars/avatar-id/original"

var (
	testOtherUserID   = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a002")
	testAvatarID      = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")
	testOtherAvatarID = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a004")
)

// TestNewAvatarUseCase_RejectsNilDependencies проверяет обязательность зависимостей.
func TestNewAvatarUseCase_RejectsNilDependencies(t *testing.T) {
	validUserRepo := &avatarUserRepositoryFake{}
	validAvatarRepo := &avatarRepositoryFake{}
	validFileStore := &fileStoreFake{}
	validMessagePublisher := &avatarMessagePublisherFake{}

	tests := []struct {
		name             string
		userRepo         avatarUserRepository
		avatarRepo       avatarRepository
		fileStore        fileStorage
		messagePublisher avatarMessagePublisher
		wantErr          string
	}{
		{
			name:             "user repository",
			avatarRepo:       validAvatarRepo,
			fileStore:        validFileStore,
			messagePublisher: validMessagePublisher,
			wantErr:          "user repository is not provided",
		},
		{
			name:             "avatar repository",
			userRepo:         validUserRepo,
			fileStore:        validFileStore,
			messagePublisher: validMessagePublisher,
			wantErr:          "avatar repository is not provided",
		},
		{
			name:             "file storage",
			userRepo:         validUserRepo,
			avatarRepo:       validAvatarRepo,
			messagePublisher: validMessagePublisher,
			wantErr:          "file storage is not provided",
		},
		{
			name:       "message publisher",
			userRepo:   validUserRepo,
			avatarRepo: validAvatarRepo,
			fileStore:  validFileStore,
			wantErr:    "avatar message publisher is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			useCase, err := NewAvatarUseCase(
				tt.userRepo,
				tt.avatarRepo,
				tt.fileStore,
				tt.messagePublisher,
			)

			// Assert
			require.EqualError(t, err, tt.wantErr)
			assert.Nil(t, useCase)
		})
	}
}

func validUploadAvatarInput() UploadAvatarInput {
	return UploadAvatarInput{
		UserID:   testUserID,
		FileName: "avatar.png",
		MIMEType: model.MIMEPNG,
		Content:  []byte("image content"),
	}
}

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
